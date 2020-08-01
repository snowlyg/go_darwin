//go:generate go run -tags generate protocol/router/gen.go
package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/jander/golog/logger"
	"github.com/kardianos/service"
	"github.com/snowlyg/go_darwin/client"
	"github.com/snowlyg/go_darwin/configure"
	"github.com/snowlyg/go_darwin/models"
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

func startHls() *hls.Server {
	hlsAddr := configure.Config.GetString("hls_addr")
	hlsListen, err := net.Listen("tcp", hlsAddr)
	if err != nil {
		logger.Error(err)
	}

	hlsServer := hls.NewServer()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Println("HLS server panic: ", r)
			}
		}()
		logger.Println("HLS listen On ", hlsAddr)
		hlsServer.Serve(hlsListen)
	}()
	return hlsServer
}

var rtmpAddr string

func startRtmp(stream *rtmp.RtmpStream, hlsServer *hls.Server) {
	rtmpAddr = configure.Config.GetString("rtmp_addr")

	rtmpListen, err := net.Listen("tcp", rtmpAddr)
	if err != nil {
		logger.Error(err)
	}

	var rtmpServer *rtmp.Server

	if hlsServer == nil {
		rtmpServer = rtmp.NewRtmpServer(stream, nil)
		logger.Println("HLS server disable....")
	} else {
		rtmpServer = rtmp.NewRtmpServer(stream, hlsServer)
		logger.Println("HLS server enable....")
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Println("RTMP server panic: ", r)
		}
	}()
	logger.Println("RTMP Listen On ", rtmpAddr)
	rtmpServer.Serve(rtmpListen)
}

func startHTTPFlv(stream *rtmp.RtmpStream) {
	httpflvAddr := configure.Config.GetString("httpflv_addr")

	flvListen, err := net.Listen("tcp", httpflvAddr)
	if err != nil {
		logger.Error(err)
	}

	hdlServer := httpflv.NewServer(stream)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Println("HTTP-FLV server panic: ", r)
			}
		}()
		logger.Println("HTTP-FLV listen On ", httpflvAddr)
		hdlServer.Serve(flvListen)
	}()
}

func startAPI(stream *rtmp.RtmpStream) {
	apiAddr := configure.Config.GetString("api_addr")
	if apiAddr != "" {
		opListen, err := net.Listen("tcp", apiAddr)
		if err != nil {
			logger.Println(err)
		}
		opServer := router.NewServer(stream, rtmpAddr)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Println("HTTP-API server panic: ", r)
				}
			}()
			logger.Println("HTTP-API listen On ", apiAddr)
			opServer.Serve(opListen)
		}()
	}
}

func startGo() {
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
					args := []string{"-re", "-rtsp_transport", "tcp", "-i", fmt.Sprintf("%s", pusher.Source), "-c", "copy", "-f", "flv", fmt.Sprintf("rtmp://%s:1935/godarwin/%s", "127.0.0.1", pusher.Key)}
					if strings.Contains(pusher.Source, "rtmp") {
						args = []string{"-re", "-i", fmt.Sprintf("%s", pusher.Source), "-c", "copy", "-f", "flv", fmt.Sprintf("rtmp://%s:1935/godarwin/%s", "127.0.0.1", pusher.Key)}
					}

					logger.Println(args)

					cmdOptions := cmd.Options{
						Buffered:  true,
						Streaming: true,
					}

					envCmd := cmd.NewCmdOptions(cmdOptions, "ffmpeg", args...)
					doneChan := make(chan struct{})
					go func() {
						defer close(doneChan)
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

					<-envCmd.Start()
					<-doneChan
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

type program struct{}

func (p *program) Start(s service.Service) error {
	// 非阻塞启动。异步执行
	go p.run()
	return nil
}
func (p *program) run() {
	// 执行内容
	err := models.Init()
	if err != nil {
		return
	}

	stream := rtmp.NewRtmpStream()
	hlsServer := startHls()
	go startGo()
	go startHTTPFlv(stream)
	go startAPI(stream)
	go startRtmp(stream, hlsServer)
}

func (p *program) Stop(s service.Service) error {
	return nil
}

func main() {
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
		DisplayName: "视频监控管理平台",
		Description: "视频监控管理平台，支持 RTSP,RTMP,FLV,M3U8",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Println(err)
	}

	if len(os.Args) == 2 {
		if os.Args[1] == "install" {
			err := s.Install()
			if err != nil {
				panic(err)
			}
			logger.Println("服务安装成功")
			return
		}

		if os.Args[1] == "remove" {
			err := s.Uninstall()
			if err != nil {
				panic(err)
			}
			logger.Println("服务卸载成功")
			return
		}

		if os.Args[1] == "start" {
			err := s.Start()
			if err != nil {
				panic(err)
			}
			logger.Println("服务启动成功")
			return
		}

		if os.Args[1] == "stop" {
			err := s.Stop()
			if err != nil {
				panic(err)
			}
			logger.Println("服务停止成功")
			return
		}

		if os.Args[1] == "restart" {
			err := s.Restart()
			if err != nil {
				panic(err)
			}

			logger.Println("服务重启成功")
			return
		}

		if os.Args[1] == "version" {
			fmt.Println(fmt.Sprintf("版本号：%s", Version))
			return
		}

	}
	err = s.Run()
	if err != nil {
		logger.Println(err)
	}
}
