package main

// Config defines the config for plugin clickhouse
type Config struct {
	DSN       string `yaml:"dsn"`
	BatchSize int    `yaml:"batchSize"`
}
