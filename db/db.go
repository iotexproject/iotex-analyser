package db

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	newLogger := logger.Discard
	if config.Default.Database.Debug {
		newLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second,   // Slow SQL threshold
				LogLevel:                  logger.Silent, // Log level
				IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
				Colorful:                  false,         // Disable color
			},
		)
	}
	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   false,
		PrepareStmt:                              true,
		Logger:                                   newLogger,
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

	if config.Default.Database.Debug {
		db = db.Debug()
	}
	db.AutoMigrate(&IndexHeight{}, &Store{})
	return db, err
}

func DB() *gorm.DB {
	return db
}

// AutoMigrate run auto migration for given models
func AutoMigrate(index string, dst ...interface{}) error {
	height, err := GetIndexHeight(index)
	if err != nil {
		return err
	}
	if height == 0 {
		err = db.Migrator().DropTable(dst...)
		if err != nil {
			return err
		}
		return db.Migrator().CreateTable(dst...)
	}
	return nil
}

func LoadDBFromEnv() (*gorm.DB, error) {
	_, err := config.New(os.Getenv("ConfigPath"))
	if err != nil {
		return nil, err
	}
	return Connect()
}
