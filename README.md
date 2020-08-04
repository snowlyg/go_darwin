# go_darwin

#### 后端参考项目
[livego](https://github.com/gwuhaolin/livego)

#### 注意
- 需要安装 ffmepg

####
```shell script
.\go_darwin.exe install
.\go_darwin.exe start
.\go_darwin.exe stop
.\go_darwin.exe restart
.\go_darwin.exe uninstall
```

#### 前端文档
[vue-element-admin-site](https://panjiachen.github.io/vue-element-admin-site/guide)

#### air
[cosmtrek/air](https://github.com/cosmtrek/air)

##### 播放地址
- RTMP:rtmp://localhost:1935/godarwin/movie
- FLV:http://127.0.0.1:7001/godarwin/movie.flv
- HLS:http://127.0.0.1:7002/godarwin/movie.m3u8

####
go build -a -installsuffix cgo -ldflags "-w -s -X main.Version=v1.0"
