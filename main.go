//go:generate go run -tags generate protocol/api/gen.go
package main

import (
	"fmt"
	"net"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/kardianos/service"
	log "github.com/sirupsen/logrus"
	"github.com/snowlyg/go_darwin/client"
	"github.com/snowlyg/go_darwin/configure"
	"github.com/snowlyg/go_darwin/models"
	"github.com/snowlyg/go_darwin/protocol/api"
	"github.com/snowlyg/go_darwin/protocol/hls"
	"github.com/snowlyg/go_darwin/protocol/httpflv"
	"github.com/snowlyg/go_darwin/protocol/rtmp"
)

var VERSION = "master"
var logger service.Logger

func startHls() *hls.Server {
	hlsAddr := configure.Config.GetString("hls_addr")
	hlsListen, err := net.Listen("tcp", hlsAddr)
	if err != nil {
		log.Fatal(err)
	}

	hlsServer := hls.NewServer()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("HLS server panic: ", r)
			}
		}()
		log.Info("HLS listen On ", hlsAddr)
		hlsServer.Serve(hlsListen)
	}()
	return hlsServer
}

var rtmpAddr string

func startRtmp(stream *rtmp.RtmpStream, hlsServer *hls.Server) {
	rtmpAddr = configure.Config.GetString("rtmp_addr")

	rtmpListen, err := net.Listen("tcp", rtmpAddr)
	if err != nil {
		log.Fatal(err)
	}

	var rtmpServer *rtmp.Server

	if hlsServer == nil {
		rtmpServer = rtmp.NewRtmpServer(stream, nil)
		log.Info("HLS server disable....")
	} else {
		rtmpServer = rtmp.NewRtmpServer(stream, hlsServer)
		log.Info("HLS server enable....")
	}

	defer func() {
		if r := recover(); r != nil {
			log.Error("RTMP server panic: ", r)
		}
	}()
	log.Info("RTMP Listen On ", rtmpAddr)
	rtmpServer.Serve(rtmpListen)
}

func startHTTPFlv(stream *rtmp.RtmpStream) {
	httpflvAddr := configure.Config.GetString("httpflv_addr")

	flvListen, err := net.Listen("tcp", httpflvAddr)
	if err != nil {
		log.Fatal(err)
	}

	hdlServer := httpflv.NewServer(stream)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("HTTP-FLV server panic: ", r)
			}
		}()
		log.Info("HTTP-FLV listen On ", httpflvAddr)
		hdlServer.Serve(flvListen)
	}()
}

func startAPI(stream *rtmp.RtmpStream) {
	apiAddr := configure.Config.GetString("api_addr")

	if apiAddr != "" {
		opListen, err := net.Listen("tcp", apiAddr)
		if err != nil {
			log.Fatal(err)
		}
		opServer := api.NewServer(stream, rtmpAddr)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error("HTTP-API server panic: ", r)
				}
			}()
			log.Info("HTTP-API listen On ", apiAddr)
			opServer.Serve(opListen)
		}()
	}
}

func startGo() {
	server := client.GetServer()
	go func() {
		log.Println("start go")
		pusher2ffmpegMap := make(map[*client.Pusher]*cmd.Cmd)
		var pusher *client.Pusher
		addChnOk := true
		removeChnOk := true
		for addChnOk || removeChnOk {
			select {
			case pusher, addChnOk = <-server.AddPusherCh:

				log.Debugln("AddPusherCh:", pusher)
				if addChnOk {
					args := []string{"-re", "-rtsp_transport", "tcp", "-i", fmt.Sprintf("%s", "rtsp://222.133.29.218:9090/dss/monitor/param?cameraid=1000096%240&substream=1"), "-c", "copy", "-f", "flv", fmt.Sprintf("rtmp://%s:1935/godarwin/%s", "localhost", pusher.Key)}
					cmdOptions := cmd.Options{
						Buffered:  false,
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
								log.Println(line)
							case line, open := <-envCmd.Stderr:
								if !open {
									envCmd.Stderr = nil
									continue
								}
								fmt.Fprintln(os.Stderr, line)
							}
						}
					}()

					<-envCmd.Start()
					<-doneChan
				} else {
					log.Printf("addPusherChan closed")
				}
			case pusher, removeChnOk = <-server.RemovePusherCh:
				if removeChnOk {
					cmd := pusher2ffmpegMap[pusher]
					err := cmd.Stop()
					if err != nil {
						log.Printf("prepare to SIGTERM to process:%v", err)
					}
					delete(pusher2ffmpegMap, pusher)
					log.Printf("delete ffmpeg from pull stream from pusher[%v]", pusher)
				} else {
					for _, cmd := range pusher2ffmpegMap {
						err := cmd.Stop()
						if err != nil {
							log.Printf("prepare to SIGTERM to process:%v", err)
						}
					}
					pusher2ffmpegMap = make(map[*client.Pusher]*cmd.Cmd)
					log.Printf("removePusherChan closed")
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
	startGo()
	hlsServer := startHls()
	startHTTPFlv(stream)
	startAPI(stream)
	startRtmp(stream, hlsServer)
}

func (p *program) Stop(s service.Service) error {
	return nil
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf(" %s:%d", filename, f.Line)
		},
	})
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("live panic: ", r)
			time.Sleep(1 * time.Second)
		}
	}()

	log.Infof(` 
====================================================
  ____  ___  ____    _    ______        _____ _   _ 
 / ___|/ _ \|  _ \  / \  |  _ \ \      / /_ _| \ | |
| |  _| | | | | | |/ _ \ | |_) \ \ /\ / / | ||  \| |
| |_| | |_| | |_| / ___ \|  _ < \ V  V /  | || |\  |
 \____|\___/|____/_/   \_\_| \_\ \_/\_/  |___|_| \_|

====================================================

version: %s
	`, VERSION)

	svcConfig := &service.Config{
		Name:        "godarwin",
		DisplayName: "视频监控管理平台",
		Description: "视频监控管理平台，支持 RTSP,RTMP,FLV,M3U8",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 {
		err = service.Control(s, os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
