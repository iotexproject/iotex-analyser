package db

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"gorm.io/driver/postgres"
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
	case "postgres":
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", host, user, password, name, port)
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)
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

// SetDB overrides the global DB connection, intended for testing only.
func SetDB(gormDB *gorm.DB) {
	db = gormDB
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

// EnsureTables creates any of the given tables that don't already exist,
// without dropping any data. Intended for tables introduced after a plugin
// was first deployed: AutoMigrate's drop-then-recreate only runs at index
// height 0, so on upgrade new tables would otherwise never be created.
//
// Safe to call alongside AutoMigrate: at height 0 the tables AutoMigrate
// just created exist already, so EnsureTables is a no-op; at height > 0
// AutoMigrate is a no-op and EnsureTables creates only what's missing.
func EnsureTables(dst ...interface{}) error {
	for _, m := range dst {
		if db.Migrator().HasTable(m) {
			continue
		}
		if err := db.Migrator().CreateTable(m); err != nil {
			return fmt.Errorf("ensure table %T: %w", m, err)
		}
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
