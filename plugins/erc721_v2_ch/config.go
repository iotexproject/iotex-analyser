package main

import (
	ch "github.com/ClickHouse/clickhouse-go/v2"
	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
)

// Config defines the config for plugin clickhouse
type Config struct {
	DSN       string `yaml:"dsn"`
	BatchSize int    `yaml:"batchSize"`
}

func openChConn(cfg *Config) (*gorm.DB, error) {
	var err error
	opt, err := ch.ParseDSN(cfg.DSN)
	if err != nil {
		return nil, err
	}
	chConn := ch.OpenDB(opt)
	return gorm.Open(clickhouse.New(clickhouse.Config{
		Conn:                   chConn,
		DefaultTableEngineOpts: "ENGINE=MergeTree() PARTITION BY toYYYYMM(timestamp) ORDER BY block_height", // default table engine options
	}))
}
