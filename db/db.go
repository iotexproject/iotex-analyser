package db

import (
	"errors"
	"fmt"

	"github.com/iotexproject/iotex-analyser/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func Connect() (*gorm.DB, error) {
	var err error
	var dsn string

	driver := config.Default.Database.Driver
	user := config.Default.Database.User
	password := config.Default.Database.Password
	host := config.Default.Database.Host
	port := config.Default.Database.Port
	name := config.Default.Database.Name

	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	}
	switch driver {
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", user, password, host, port, name)
		db, err = gorm.Open(mysql.Open(dsn), gormConfig)
		if err != nil {
			return db, err
		}
	case "postgres":
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", host, user, password, name, port)
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)
		if err != nil {
			return db, err
		}
	case "sqlite3":
		db, err = gorm.Open(sqlite.Open(name), gormConfig)
		if err != nil {
			return db, err
		}
	default:
		err = errors.New("unsopport gorm driver: " + driver)
	}

	db.AutoMigrate(&IndexHeight{})
	return db, err
}

func DB() *gorm.DB {
	return db.Debug()
}
