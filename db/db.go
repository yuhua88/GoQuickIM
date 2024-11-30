package db

import (
	"GoQuickIM/config"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/sirupsen/logrus"
)

var dbMap = map[string]*gorm.DB{}

type DbGoChat struct {
}

var syncLock sync.Mutex

func init() {
	initDB("gochat")

}
func parseMysqlDSN(mysqlDSN config.CommonMySql) string {
	return fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", mysqlDSN.UserName, mysqlDSN.Password, mysqlDSN.Host, mysqlDSN.Db)
}
func initDB(dbName string) {
	var e error
	// if prod env , you should change mysql driver for yourself !!!
	mysqlDSN := config.Conf.Common.CommonMySql
	mysqlDSN.Db = dbName
	dsn := parseMysqlDSN(mysqlDSN)

	realPath, _ := filepath.Abs("./")
	configFilePath := realPath + "/db/gochat.sqlite3"
	use_mysql := 0

	syncLock.Lock()
	defer syncLock.Unlock()
	if use_mysql == 1 {
		dbMap[dbName], e = gorm.Open("mysql", dsn)
	} else {

		dbMap[dbName], e = gorm.Open("sqlite3", configFilePath)
	}

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
