package main

import (
	"context"

	std_ck "github.com/ClickHouse/clickhouse-go/v2"
	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
)

// Config defines the config for plugin clickhouse
type Config struct {
	DSN       string `yaml:"dsn"`
	PreDelete bool   `yaml:"preDelete"`
}

var chDB *gorm.DB

func openChConn(ctx context.Context, cfg *Config) error {
	var err error
	opt, err := std_ck.ParseDSN(cfg.DSN)
	if err != nil {
		return err
	}
	chConn := std_ck.OpenDB(opt)
	chDB, err = gorm.Open(clickhouse.New(clickhouse.Config{
		Conn:                   chConn,
		DefaultTableEngineOpts: "ENGINE=MergeTree() PARTITION BY toYYYYMM(timestamp) ORDER BY block_height", // default table engine options
	}))
	return err
}
