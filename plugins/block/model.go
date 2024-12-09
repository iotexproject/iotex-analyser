package main

import (
	"context"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/pkg/errors"
)

var BlockDDL = `CREATE TABLE IF NOT EXISTS block
(
    block_height UInt64 NOT NULL,
    block_hash FixedString(64) NOT NULL,
    producer_address FixedString(41) NOT NULL,
    num_actions UInt64 NOT NULL,
    timestamp DateTime64(6) NOT NULL,
    year UInt64 NOT NULL,
    month UInt64 NOT NULL,
    day UInt64 NOT NULL
)
ENGINE = ReplacingMergeTree()
PRIMARY KEY (block_height)
ORDER BY (block_height)`

type Block struct {
	BlockHeight     uint64    `ch:"block_height"`
	BlockHash       string    `ch:"block_hash"`
	ProducerAddress string    `ch:"producer_address"`
	NumActions      int       `ch:"num_actions"`
	Timestamp       time.Time `ch:"timestamp"`
	Year            int       `ch:"year"`
	Month           int       `ch:"month"`
	Day             int       `ch:"day"`
}

func (Block) TableName() string {
	return "block"
}

func (b blockPlugin) migrateTable(ctx context.Context) error {
	err := db.ChConn().Exec(ctx, BlockDDL)
	return errors.Wrapf(err, "failed to create table %s", Block{}.TableName())
}
