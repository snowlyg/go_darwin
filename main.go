//go:generate go run -tags generate protocol/router/gen.go
package main

import (
	"fmt"
	"github.com/snowlyg/go_darwin/models"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/jander/golog/logger"
	"github.com/kardianos/service"
	"github.com/snowlyg/go_darwin/client"
	"github.com/snowlyg/go_darwin/configure"
	"github.com/snowlyg/go_darwin/protocol/hls"
	"github.com/snowlyg/go_darwin/protocol/httpflv"
	"github.com/snowlyg/go_darwin/protocol/router"
	"github.com/snowlyg/go_darwin/protocol/rtmp"
	"github.com/snowlyg/go_darwin/utils"
)

func init() {
	rotatingHandler := logger.NewRotatingHandler(utils.LogDir(), "log.log", 4, 4*1024*1024)
	logger.SetHandlers(logger.Console, rotatingHandler)
}

var Version = "master"

func (p *program) startHls() {
	hlsAddr := configure.Config.HlsAddr
	if len(hlsAddr) == 0 {
		hlsAddr = "0.0.0.0:7002"
	}
	hlsListen, err := net.Listen("tcp", hlsAddr)
	if err != nil {
		logger.Error(err)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Println("HLS server panic: ", r)
			}
		}()
		logger.Println("HLS listen On ", hlsAddr)
		p.hlsServer.Serve(hlsListen)
	}()
}

var rtmpAddr string

func (p *program) startRtmp() {
	rtmpAddr = configure.Config.RtmpAddr
	//if len(rtmpAddr) == 0{
	//	rtmpAddr = "0.0.0.0:1935"
	//}

	rtmpListen, err := net.Listen("tcp", rtmpAddr)
	if err != nil {
		logger.Error(err)
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Println("RTMP server panic: ", r)
		}
	}()
	logger.Println("RTMP Listen On ", rtmpAddr)
	p.rtmpServer.Serve(rtmpListen)
}

func (p *program) startHTTPFlv() {
	httpflvAddr := configure.Config.HttpFlvAddr
	//if len(httpflvAddr) == 0{
	//	httpflvAddr = "0.0.0.0:7001"
	//}

	flvListen, err := net.Listen("tcp", httpflvAddr)
	if err != nil {
		logger.Error(err)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Println("HTTP-FLV server panic: ", r)
			}
		}()
		logger.Println("HTTP-FLV listen On ", httpflvAddr)
		p.flvServer.Serve(flvListen)
	}()
}

func (p *program) startAPI() {
	apiAddr := configure.Config.ApiAddr
	//if len(apiAddr) == 0{
	//	apiAddr = "127.0.0.1:8090"
	//}
	if apiAddr != "" {
		opListen, err := net.Listen("tcp", apiAddr)
		if err != nil {
			logger.Println(err)
		}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Println("HTTP-API server panic: ", r)
				}
			}()
			logger.Println("HTTP-API listen On ", apiAddr)
			p.apiServer.Serve(opListen)
		}()
	}
}

func (p *program) startGo() {
	server := client.GetServer()
	go func() {
		logger.Println("start go")
		pusher2ffmpegMap := make(map[*client.Pusher]*cmd.Cmd)
		var pusher *client.Pusher
		addChnOk := true
		removeChnOk := true
		for addChnOk || removeChnOk {
			select {
			case pusher, addChnOk = <-server.AddPusherCh:
				logger.Println("AddPusherCh:", pusher)
				if addChnOk {
					args := []string{"-fflags", "genpts", "-re", "-rtsp_transport", "tcp", "-i", fmt.Sprintf("%s", pusher.Source), "-c", "copy", "-f", "flv", fmt.Sprintf("rtmp://%s:1935/godarwin/%s", "127.0.0.1", pusher.Key)}
					if strings.Contains(pusher.Source, "rtmp") {
						args = []string{"-re", "-i", fmt.Sprintf("%s", pusher.Source), "-c", "copy", "-f", "flv", fmt.Sprintf("rtmp://%s:1935/godarwin/%s", "127.0.0.1", pusher.Key)}
					}
					logger.Println(args)
					cmdOptions := cmd.Options{
						Buffered:  true,
						Streaming: true,
					}

					envCmd := cmd.NewCmdOptions(cmdOptions, "ffmpeg", args...)
					go func() {
						for envCmd.Stdout != nil || envCmd.Stderr != nil {
							select {
							case line, open := <-envCmd.Stdout:
								if !open {
									envCmd.Stdout = nil
									continue
								}
								logger.Println(line)
							case line, open := <-envCmd.Stderr:
								if !open {
									envCmd.Stderr = nil
									continue
								}
								logger.Println(os.Stderr, line)
							}
						}
					}()

					envCmd.Start()
				} else {
					logger.Printf("addPusherChan closed")
				}
			case pusher, removeChnOk = <-server.RemovePusherCh:
				if removeChnOk {
					goCmd := pusher2ffmpegMap[pusher]
					if goCmd != nil {
						err := goCmd.Stop()
						if err != nil {
							logger.Printf("prepare to SIGTERM to process:%v", err)
						}
						delete(pusher2ffmpegMap, pusher)
					}

					logger.Printf("delete ffmpeg from pull stream from pusher[%v]", pusher)
				} else {
					for _, goCmd := range pusher2ffmpegMap {
						if goCmd != nil {
							err := goCmd.Stop()
							if err != nil {
								logger.Printf("prepare to SIGTERM to process:%v", err)
							}
						}
					}
					pusher2ffmpegMap = make(map[*client.Pusher]*cmd.Cmd)
					logger.Printf("removePusherChan closed")
				}
			}
		}
	}()
}

type program struct {
	hlsServer  *hls.Server
	flvServer  *httpflv.Server
	apiServer  *router.Server
	rtmpServer *rtmp.Server
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}
func (p *program) run() {
	// 执行内容
	err := models.Init()
	if err != nil {
		return
	}

	p.startGo()
	p.startHTTPFlv()
	p.startAPI()
	p.startHls()
	p.startRtmp()
}

func (p *program) Stop(s service.Service) error {
	models.Close()
	return nil
}

func main() {

	fmt.Println(configure.Config)
	defer func() {
		if r := recover(); r != nil {
			logger.Println("live panic: ", r)
			time.Sleep(1 * time.Second)
		}
	}()

	logger.Println(fmt.Sprintf(` 
====================================================
  ____  ___  ____    _    ______        _____ _   _ 
 / ___|/ _ \|  _ \  / \  |  _ \ \      / /_ _| \ | |
| |  _| | | | | | |/ _ \ | |_) \ \ /\ / / | ||  \| |
| |_| | |_| | |_| / ___ \|  _ < \ V  V /  | || |\  |
 \____|\___/|____/_/   \_\_| \_\ \_/\_/  |___|_| \_|

====================================================

version: %s`, Version))

	svcConfig := &service.Config{
		Name:        "godarwin",
		DisplayName: "Go Service Example",
		Description: "This is an example Go service.",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Println(err)
	}

	stream := rtmp.NewRtmpStream()
	apiAddr := configure.Config.ApiAddr
	if apiAddr != "" {
		opServer := router.NewServer(stream, rtmpAddr)
		prg.apiServer = opServer
	}

	hdlServer := httpflv.NewServer(stream)
	prg.flvServer = hdlServer

	hlsServer := hls.NewServer()
	var rtmpServer *rtmp.Server
	if hlsServer == nil {
		rtmpServer = rtmp.NewRtmpServer(stream, nil)
		logger.Println("HLS server disable....")
	} else {
		rtmpServer = rtmp.NewRtmpServer(stream, hlsServer)
		logger.Println("HLS server enable....")
	}

	prg.rtmpServer = rtmpServer
	prg.hlsServer = hlsServer

	if len(os.Args) == 2 {

		if os.Args[1] == "version" {
			logger.Println(fmt.Sprintf("版本号：%s", Version))
			return
		}

		err = service.Control(s, os.Args[1])
		if err != nil {
			logger.Fatal(err)
		}
		return

	}
	err = s.Run()
	if err != nil {
		logger.Println(err)
	}
}
