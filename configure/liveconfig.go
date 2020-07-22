package configure

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/kr/pretty"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

/*
{
  "server": [
    {
      "appname": "live",
      "live": true,
	  "hls": true,
	  "static_push": []
    }
  ]
}
*/

type Application struct {
	Appname    string   `mapstructure:"appname"`
	Live       bool     `mapstructure:"live"`
	Hls        bool     `mapstructure:"hls"`
	StaticPush []string `mapstructure:"static_push"`
}

type Applications []Application

type JWT struct {
	Secret    string `mapstructure:"secret"`
	Algorithm string `mapstructure:"algorithm"`
}
type ServerCfg struct {
	Level           string       `mapstructure:"level"`
	ConfigFile      string       `mapstructure:"config_file"`
	FLVDir          string       `mapstructure:"flv_dir"`
	RTMPAddr        string       `mapstructure:"rtmp_addr"`
	HTTPFLVAddr     string       `mapstructure:"httpflv_addr"`
	HLSAddr         string       `mapstructure:"hls_addr"`
	HLSKeepAfterEnd bool         `mapstructure:"hls_keep_after_end"`
	APIAddr         string       `mapstructure:"api_addr"`
	RedisAddr       string       `mapstructure:"redis_addr"`
	RedisPwd        string       `mapstructure:"redis_pwd"`
	ReadTimeout     int          `mapstructure:"read_timeout"`
	WriteTimeout    int          `mapstructure:"write_timeout"`
	GopNum          int          `mapstructure:"gop_num"`
	JWT             JWT          `mapstructure:"jwt"`
	Server          Applications `mapstructure:"server"`
}

// default config
var defaultConf = ServerCfg{
	Level:           "debug",
	ConfigFile:      "godarwin.yaml",
	RTMPAddr:        ":1935",
	HTTPFLVAddr:     ":7001",
	HLSAddr:         ":7002",
	HLSKeepAfterEnd: false,
	APIAddr:         ":8090",
	WriteTimeout:    10,
	ReadTimeout:     10,
	GopNum:          1,
	Server: Applications{{
		Appname:    "godarwin",
		Live:       true,
		Hls:        true,
		StaticPush: nil,
	}},
}

var Config = viper.New()

func initLog() {
	if l, err := log.ParseLevel(Config.GetString("level")); err == nil {
		log.SetLevel(l)
		log.SetReportCaller(l == log.DebugLevel)
	}
}

func init() {
	defer Init()

	// 默认配置
	b, err := json.Marshal(defaultConf)
	if err != nil {
		log.Error(err)
	}
	defaultConfig := bytes.NewReader(b)
	viper.SetConfigType("json")
	err = viper.ReadConfig(defaultConfig)
	if err != nil {
		log.Error(err)
	}

	err = Config.MergeConfigMap(viper.AllSettings())
	if err != nil {
		log.Error(err)
	}

	// 文件
	configFile := Config.GetString("configfile")
	if len(configFile) > 0 {
		Config.SetConfigFile(configFile)
		Config.AddConfigPath(".")
		err = Config.ReadInConfig()
		if err != nil {
			log.Warning(err)
			log.Info("使用默认配置")
		} else {
			err := Config.MergeInConfig()
			if err != nil {
				log.Error(err)
			}
		}
	}

	// 环境变量
	replacer := strings.NewReplacer(".", "_")
	Config.SetEnvKeyReplacer(replacer)
	Config.AllowEmptyEnv(true)
	Config.AutomaticEnv()

	// Log
	initLog()

	// 打印最终配置
	c := ServerCfg{}
	Config.Unmarshal(&c)
	log.Debugf("Current configurations: \n%# v", pretty.Formatter(c))
}

func CheckAppName(appname string) bool {
	apps := Applications{}
	Config.UnmarshalKey("server", &apps)
	for _, app := range apps {
		if app.Appname == appname {
			return app.Live
		}
	}
	return false
}

func GetStaticPushUrlList(appname string) ([]string, bool) {
	apps := Applications{}
	Config.UnmarshalKey("server", &apps)
	for _, app := range apps {
		if (app.Appname == appname) && app.Live {
			if len(app.StaticPush) > 0 {
				return app.StaticPush, true
			} else {
				return nil, false
			}
		}
	}
	return nil, false
}
