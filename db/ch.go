package db

import (
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/pkg/errors"
)

var (
	defaultChConn driver.Conn
)

// ConnectClickhouse connects to clickhouse
func ConnectClickhouse(dsn string) (driver.Conn, error) {
	op, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse clickhouse dsn")
	}
	chConn, err := clickhouse.Open(op)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect clickhouse")
	}
	defaultChConn = chConn
	return chConn, nil
}

func ChConn() driver.Conn {
	return defaultChConn
}
