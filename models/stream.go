package models

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/snowlyg/go_darwin/configure"
	"github.com/snowlyg/go_darwin/utils"
	"github.com/snowlyg/go_darwin/utils/db"
	"github.com/snowlyg/go_darwin/utils/uid"
	"strconv"
)

type Stream struct {
	gorm.Model
	Status   bool
	RoomName string `gorm:"type:varchar(256);unique"`
	Key      string `gorm:"type:varchar(256)"`
	Source   string `gorm:"type:varchar(256)"`
	FlvUrl   string `gorm:"type:varchar(256)"`
	HlsUrl   string `gorm:"type:varchar(256)"`
	RtmpUrl  string `gorm:"type:varchar(256)"`
}

func GetStreams(offset, limit int) ([]*Stream, int64) {
	var streams []*Stream
	var count int64
	db.SQLite.Find(&streams).Count(&count)
	db.SQLite.Offset(offset).Limit(limit).Order("id desc").Find(&streams)
	return streams, count
}

func GetStream(Sid string) (*Stream, error) {
	id, err := strconv.ParseUint(Sid, 10, 64)
	if err != nil {
		return nil, err
	}

	stream := Stream{}
	db.SQLite.Where("id = ?", id).First(&stream)
	return &stream, nil
}

func AddStream(source string) (*Stream, error) {
	roomName := utils.MD5(uid.NewId())
	flvUrl := fmt.Sprintf("http://%s/godarwin/%s.flv", configure.Config.PlayFlvAddr, roomName)
	hlsUrl := fmt.Sprintf("http://%s/godarwin/%s.m3u8", configure.Config.PlayHlsAddr, roomName)
	rtmpUrl := fmt.Sprintf("rtmp://%s/godarwin/%s", configure.Config.PlayRtmpAddr, roomName)
	stream := Stream{
		Status:   false,
		RoomName: roomName,
		Source:   source,
		FlvUrl:   flvUrl,
		HlsUrl:   hlsUrl,
		RtmpUrl:  rtmpUrl,
	}
	if err := db.SQLite.Create(&stream).Error; err != nil {
		return nil, err
	}
	return &stream, nil
}

func UpdateStream(id uint, source string) (*Stream, error) {
	stream := new(Stream)
	if db.SQLite.Where("id = ?", id).First(stream).RecordNotFound() {
		return nil, errors.New(fmt.Sprintf("拉流数据不存在, key：%v", stream))
	}
	flvUrl := fmt.Sprintf("http://%s/godarwin/%s.flv", configure.Config.PlayFlvAddr, stream.RoomName)
	hlsUrl := fmt.Sprintf("http://%s/godarwin/%s.m3u8", configure.Config.PlayHlsAddr, stream.RoomName)
	rtmpUrl := fmt.Sprintf("rtmp://%s/godarwin/%s", configure.Config.PlayRtmpAddr, stream.RoomName)

	stream.Status = false
	stream.Source = source
	stream.FlvUrl = flvUrl
	stream.HlsUrl = hlsUrl
	stream.RtmpUrl = rtmpUrl
	db.SQLite.Save(stream)

	fmt.Println(stream, flvUrl)

	return stream, nil
}

func StartStream(id uint) (*Stream, error) {
	stream := new(Stream)
	if db.SQLite.Where("id = ?", id).First(stream).RecordNotFound() {
		return nil, errors.New(fmt.Sprintf("拉流数据不存在, key：%v", stream))
	}

	key, err := configure.RoomKeys.GetKey(stream.RoomName)
	if err != nil {
		return nil, err
	}

	stream.Status = true
	stream.Key = key
	db.SQLite.Save(stream)

	return stream, nil
}

func StopStream(id uint) (*Stream, error) {
	stream := new(Stream)
	if db.SQLite.Where("id = ?", id).First(stream).RecordNotFound() {
		return nil, errors.New(fmt.Sprintf("拉流数据不存在, key：%v", stream))
	}

	stream.Status = false
	stream.Key = ""
	db.SQLite.Save(stream)

	return stream, nil
}

func DeleteStream(id uint) error {
	stream := Stream{Model: gorm.Model{ID: id}}
	if err := db.SQLite.Delete(&stream).Error; err != nil {
		return err
	}
	return nil
}
