package models

import (
	"github.com/jinzhu/gorm"
	"github.com/snowlyg/go_darwin/configure"
	"github.com/snowlyg/go_darwin/utils/db"
	"github.com/snowlyg/go_darwin/utils/uid"

	"strconv"
)

type Stream struct {
	gorm.Model
	Status   bool
	PusherId string `gorm:"type:varchar(256)"`
	RoomName string `gorm:"type:varchar(256);unique"`
	Key      string `gorm:"type:varchar(256);unique"`
	Source   string `gorm:"type:varchar(256)"`
}

func GetStreams(offset, limit int) ([]*Stream, int) {
	var streams []*Stream
	var count int
	db.SQLite.Find(&streams).Count(&count)
	db.SQLite.Offset(offset).Limit(limit).Order("id desc").Find(&streams)
	return streams, count
}

func GetStream(Sid string) Stream {
	id, _ := strconv.ParseUint(Sid, 10, 64)

	stream := Stream{}
	db.SQLite.Where("id = ?", id).First(&stream)
	return stream
}

func AddStream(source, roomName string) *Stream {
	stream := Stream{Status: false, PusherId: uid.NewId(), RoomName: roomName, Source: source}
	db.SQLite.Create(&stream)
	return &stream
}

func UpdateStream(Sid, roomName, source string) {
	id, _ := strconv.ParseUint(Sid, 10, 64)
	stream := Stream{Model: gorm.Model{ID: uint(id)}}
	db.SQLite.Model(&stream).Updates(Stream{RoomName: roomName, Source: source})
}

func StartStream(Sid string) *Stream {
	id, _ := strconv.ParseUint(Sid, 10, 64)
	stream := &Stream{Model: gorm.Model{ID: uint(id)}}

	key, _ := configure.RoomKeys.GetKey(stream.RoomName)

	db.SQLite.First(&stream)
	stream.Status = true
	stream.Key = key
	db.SQLite.Save(&stream)

	return stream
}

func StopStream(Sid string) {
	id, _ := strconv.ParseUint(Sid, 10, 64)
	stream := &Stream{Model: gorm.Model{ID: uint(id)}}
	db.SQLite.First(&stream)
	stream.Status = false
	stream.Key = ""
	db.SQLite.Save(&stream)
}

func DeleteStream(Sid string) {
	id, _ := strconv.ParseUint(Sid, 10, 64)
	stream := Stream{Model: gorm.Model{ID: uint(id)}}
	db.SQLite.Delete(&stream)
}
