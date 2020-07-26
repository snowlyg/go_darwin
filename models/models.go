package models

import (
	"github.com/snowlyg/go_darwin/utils"
	"github.com/snowlyg/go_darwin/utils/db"
)

func Init() (err error) {
	err = db.Init()
	if err != nil {
		return
	}
	db.SQLite.AutoMigrate(User{}, Stream{})
	count := 0

	defUser := "admin"
	defPass := "admin"
	db.SQLite.Model(User{}).Where("username = ?", defUser).Count(&count)
	if count == 0 {
		db.SQLite.Create(&User{
			Username: defUser,
			Password: utils.MD5(defPass),
		})
	}
	return
}

func Close() {
	db.Close()
}
