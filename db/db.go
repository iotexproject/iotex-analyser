package db

import (
	"database/sql"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/pkg/log"
)

var conn *sql.DB
var once sync.Once

func GetDB() *sql.DB {
	once.Do(func() {
		var err error
		if conn, err = sql.Open("mysql", config.Default.Database.Dsn); err != nil {
			log.L().Fatal(err.Error())
		}
		conn.SetMaxOpenConns(20) // Sane default
		conn.SetMaxIdleConns(0)
		conn.SetConnMaxLifetime(time.Nanosecond)
	})
	return conn
}
