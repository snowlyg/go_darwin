package models

import (
	"errors"
	"fmt"
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
	Key      string `gorm:"type:varchar(256)"`
	Source   string `gorm:"type:varchar(256)"`
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

func AddStream(source, roomName string) (*Stream, error) {
	stream := Stream{Status: false, PusherId: uid.NewId(), RoomName: roomName, Source: source}
	if err := db.SQLite.Create(&stream).Error; err != nil {
		return nil, err
	}
	return &stream, nil
}

func UpdateStream(Sid, roomName, source string) error {
	id, err := strconv.ParseUint(Sid, 10, 64)
	if err != nil {
		return err
	}
	stream := Stream{Model: gorm.Model{ID: uint(id)}}
	db.SQLite.Model(&stream).Updates(Stream{RoomName: roomName, Source: source})

	return nil
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
