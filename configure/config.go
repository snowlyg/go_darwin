package configure

import (
	"path/filepath"

	"github.com/jinzhu/configor"
	"github.com/snowlyg/go_darwin/utils"
)

type server struct {
	Appname    string   `default:"godarwin" env:"ServerAppname"`
	Live       bool     `default:"true" env:"ServerLive"`
	Hls        bool     `default:"true" env:"ServerHls"`
	StaticPush []string `default:"" env:"ServerStaticpush"`
}

var Config = struct {
	FlvDir          string `default:"./tmp" env:"flvdir"`
	HttpFlvAddr     string `default:"0.0.0.0:7001" env:"httpflvaddr"`
	PlayFlvAddr     string `default:"112.74.61.105:7001" env:"playflvaddr"`
	RtmpAddr        string `default:"0.0.0.0:1935" env:"rtmpaddr"`
	PlayRtmpAddr    string `default:"112.74.61.105:1935" env:"playrtmpaddr"`
	ReadTimeout     int    `default:"10" env:"readtimeout"`
	WriteTimeout    int    `default:"10" env:"writetimeout"`
	GopNum          int    `default:"1" env:"gopnum"`
	HlsAddr         string `default:"0.0.0.0:7002" env:"hlsaddr"`
	PlayHlsAddr     string `default:"112.74.61.105:7002" env:"playhlsaddr"`
	ApiAddr         string `default:"localhost:8090" env:"apiaddr"`
	HlsKeepAfterEnd bool   `default:"true" env:"hlskeepafterend"`
	Redis           struct {
		Addr string `default:"" env:"redisaddr"`
		Pwd  string `default:"" env:"redispwd"`
	}
	Server []struct {
		Appname    string   `default:"godarwin" env:"ServerAppname"`
		Live       bool     `default:"true" env:"ServerLive"`
		Hls        bool     `default:"true" env:"ServerHls"`
		StaticPush []string `default:"" env:"ServerStaticpush"`
	}
}{}

func init() {
	configPath := filepath.Join(utils.CWD(), "godarwin.yml")
	if err := configor.Load(&Config, configPath); err != nil {
		panic(err)
	}
	//默认配置
	//if Config.Server == nil {
	//	server := server{"godarwin", true, true, nil}
	//	Config.Server = append(Config.Server, &server)
	//}
}

func CheckAppName(appname string) bool {
	for _, server := range Config.Server {
		if server.Appname == appname {
			return server.Live
		}
	}
	return false
}

func GetStaticPushUrlList(appname string) ([]string, bool) {
	for _, server := range Config.Server {
		if (server.Appname == appname) && server.Live {
			if len(server.StaticPush) > 0 {
				return server.StaticPush, true
			} else {
				return nil, false
			}
		}
	}

	return nil, false
}
