//go:generate go run -tags generate protocol/router/gen.go
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/kardianos/service"
	"github.com/snowlyg/go_darwin/client"
	"github.com/snowlyg/go_darwin/configure"
	"github.com/snowlyg/go_darwin/models"
	"github.com/snowlyg/go_darwin/protocol/hls"
	"github.com/snowlyg/go_darwin/protocol/httpflv"
	"github.com/snowlyg/go_darwin/protocol/router"
	"github.com/snowlyg/go_darwin/protocol/rtmp"
)

var Version = "master"

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
				log.Println("HLS server panic: ", r)
			}
		}()
		log.Println("HLS listen On ", hlsAddr)
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
		log.Println("HLS server disable....")
	} else {
		rtmpServer = rtmp.NewRtmpServer(stream, hlsServer)
		log.Println("HLS server enable....")
	}
	defer func() {
		if r := recover(); r != nil {
			log.Println("RTMP server panic: ", r)
		}
	}()
	log.Println("RTMP Listen On ", rtmpAddr)
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
				log.Println("HTTP-FLV server panic: ", r)
			}
		}()
		log.Println("HTTP-FLV listen On ", httpflvAddr)
		hdlServer.Serve(flvListen)
	}()
}

func startAPI(stream *rtmp.RtmpStream) {
	apiAddr := configure.Config.GetString("api_addr")
	if apiAddr != "" {
		opListen, err := net.Listen("tcp", apiAddr)
		if err != nil {
			log.Println(err)
		}
		opServer := router.NewServer(stream, rtmpAddr)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("HTTP-API server panic: ", r)
				}
			}()
			log.Println("HTTP-API listen On ", apiAddr)
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
				log.Println("AddPusherCh:", pusher)
				if addChnOk {
					args := []string{"-re", "-rtsp_transport", "tcp", "-i", fmt.Sprintf("%s", pusher.Path), "-c", "copy", "-f", "flv", fmt.Sprintf("rtmp://%s:1935/godarwin/%s", "localhost", pusher.Key)}
					if strings.Contains(pusher.Path, "rtmp") {
						args = []string{"-re", "-i", fmt.Sprintf("%s", pusher.Path), "-c", "copy", "-f", "flv", fmt.Sprintf("rtmp://%s:1935/godarwin/%s", "localhost", pusher.Key)}
					}

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
					goCmd := pusher2ffmpegMap[pusher]
					if goCmd != nil {
						err := goCmd.Stop()
						if err != nil {
							log.Printf("prepare to SIGTERM to process:%v", err)
						}
						delete(pusher2ffmpegMap, pusher)
					}

					log.Printf("delete ffmpeg from pull stream from pusher[%v]", pusher)
				} else {
					for _, goCmd := range pusher2ffmpegMap {
						if goCmd != nil {
							err := goCmd.Stop()
							if err != nil {
								log.Printf("prepare to SIGTERM to process:%v", err)
							}
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
	hlsServer := startHls()
	startGo()
	startHTTPFlv(stream)
	startAPI(stream)
	startRtmp(stream, hlsServer)
}

func (p *program) Stop(s service.Service) error {
	return nil
}

//func init() {
//	// Log as JSON instead of the default ASCII formatter.
//	log.SetFormatter(&log.JSONFormatter{})
//
//	// Output to stdout instead of the default stderr
//	// Can be any io.Writer, see below for File example
//	log.SetOutput(os.Stdout)
//
//	// Only log the warning severity or above.
//	log.SetLevel(log.WarnLevel)
//}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("live panic: ", r)
			time.Sleep(1 * time.Second)
		}
	}()

	fmt.Println(fmt.Sprintf(` 
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
		log.Println(err)
	}

	if len(os.Args) > 1 {
		if os.Args[1] == "version" {
			log.Println(Version)
		}

		err = service.Control(s, os.Args[1])
		if err != nil {
			log.Println(err)
		}
		return
	}

	err = s.Run()
	if err != nil {
		log.Println(err)
	}
}
