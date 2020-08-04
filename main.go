//go:generate go run -tags generate protocol/router/gen.go
package main

import (
	"context"
	"fmt"
	"github.com/snowlyg/go_darwin/utils"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/kardianos/service"
	logger "github.com/sirupsen/logrus"
	"github.com/snowlyg/go_darwin/client"
	"github.com/snowlyg/go_darwin/configure"
	"github.com/snowlyg/go_darwin/models"
	"github.com/snowlyg/go_darwin/protocol/hls"
	"github.com/snowlyg/go_darwin/protocol/httpflv"
	"github.com/snowlyg/go_darwin/protocol/router"
	"github.com/snowlyg/go_darwin/protocol/rtmp"
)

var Version = "master"
var f *os.File
var rtmpAddr string

func init() {
	f, _ := os.OpenFile(fmt.Sprintf("%s/log.log", utils.LogDir()), os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
	logger.SetFormatter(&logger.JSONFormatter{})
	logger.SetOutput(f)
	logger.SetLevel(logger.DebugLevel)
}

func (p *program) startHls() {
	hlsAddr := configure.Config.HlsAddr
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
		err = p.hlsServer.Serve(hlsListen)
		if err != nil {
			logger.Println("HLS listen Err :", err)
		}
	}()
}

func (p *program) startRtmp() {
	rtmpAddr = configure.Config.RtmpAddr

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
	err = p.rtmpServer.Serve(rtmpListen)
	if err != nil {
		logger.Println("RTMP listen Err :", err)
	}
}

func (p *program) startHTTPFlv() {
	httpflvAddr := configure.Config.HttpFlvAddr

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
		err := p.flvServer.Serve(flvListen)
		if err != nil {
			logger.Println("HTTP-FLV listen Err :", err)
		}
	}()
}

func (p *program) startAPI() {
	apiAddr := configure.Config.ApiAddr
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
			err := p.apiServer.Serve(opListen)
			if err != nil {
				logger.Println("HTTP-API listen Err :", err)
			}
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
					args := []string{"-re", "-fflags", "genpts", "-rtsp_transport", "tcp", "-i", fmt.Sprintf("%s", pusher.Source), "-map", "0", "-c", "copy", "-f", "flv", fmt.Sprintf("rtmp://%s:1935/godarwin/%s", "127.0.0.1", pusher.Key)}
					if strings.Contains(pusher.Source, "rtsp://") {
						args = []string{"-re", "-fflags", "genpts", "-rtsp_transport", "tcp", "-i", fmt.Sprintf("%s", pusher.Source), "-map", "0", "-c", "copy", "-f", "flv", fmt.Sprintf("rtmp://%s:1935/godarwin/%s", "127.0.0.1", pusher.Key)}
					} else if strings.Contains(pusher.Source, "rtmp://") {
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

func (p *program) stopAPI() (err error) {
	if p.apiServer == nil {
		err = fmt.Errorf("HTTP Server Not Found")
		return
	}

	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	p.apiServer.App.Shutdown(ctx)
	return
}

func (p *program) Stop(s service.Service) error {
	defer logger.Println("退出服务")
	defer f.Close()

	p.stopAPI()
	models.Close()
	return nil
}

func main() {
	log.Println(fmt.Sprintf(` 
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
		DisplayName: "GoDarwin",
		Description: "音视频拉流服务",
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
