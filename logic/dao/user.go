package dao

import (
	"GoQuickIM/db"
	"errors"
	"time"
)

// using gorm deal Sql
var dbIns = db.GetDb("gochat")

type User struct {
	Id         int `gorm:"primary_key"`
	UserName   string
	Password   string
	CreateTime time.Time
	db.DbGoChat
}

func (u *User) GetUserNameByUserId(userId int) string {
	var data User
	dbIns.Table(u.TableName()).Where("id=?", userId).Take(&data)
	return data.UserName
}
func (u *User) TableName() string {
	return "user"
}

// add new user
func (u *User) Add() (userId int, err error) {
	//UserName or Password nil
	if u.UserName == "" || u.Password == "" {
		return 0, errors.New("user_name or password empty")
	}
	//UserName Exist repeat?
	oUser := u.CheckHaveUserName(u.UserName)
	if oUser.Id > 0 {
		return oUser.Id, nil
	}
	u.CreateTime = time.Now()
	if err = dbIns.Table(u.TableName()).Create(&u).Error; err != nil {
		return 0, err
	}
	return u.Id, nil
}

func (u *User) CheckHaveUserName(userName string) (data User) {
	dbIns.Table(u.TableName()).Where("user_name=?", userName).Take(&data)
	return
}
