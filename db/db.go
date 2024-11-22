package db

import (
	"GoQuickIM/config"
	"path/filepath"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sirupsen/logrus"
)

var dbMap = map[string]*gorm.DB{}

type DbGoChat struct {
}

var syncLock sync.Mutex

func init() {
	initDB("gochat")
}
func initDB(dbName string) {
	var e error
	// if prod env , you should change mysql driver for yourself !!!
	realPath, _ := filepath.Abs("./")
	configFilePath := realPath + "/db/gochat.sqlite3"
	syncLock.Lock()
	defer syncLock.Unlock()
	dbMap[dbName], e = gorm.Open("sqlite3", configFilePath)
	dbMap[dbName].DB().SetMaxIdleConns(4)
	dbMap[dbName].DB().SetMaxOpenConns(20)
	dbMap[dbName].DB().SetConnMaxLifetime(8 * time.Second)
	if config.GetMode() == "dev" {
		dbMap[dbName].LogMode(true)
	}

	if e != nil {
		logrus.Error("connect db fail:", e.Error())
	}
}

func GetDb(dbName string) (db *gorm.DB) {
	if db, ok := dbMap[dbName]; ok {
		return db
	} else {
		return nil
	}
}
