package main

import (
	"context"
	"database/sql"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
)

const VERSION = "1.0.1"

type blockPlugin struct {
	tableName string
}

func (b blockPlugin) Name() string {
	return "block"
}
func (b blockPlugin) Start(ctx context.Context) error {
	createSql := "CREATE TABLE IF NOT EXISTS `" + b.tableName + "` (" +
		"`block_height` bigint(20) NOT NULL," +
		"`block_hash` varchar(64) NOT NULL," +
		"`producer_address` varchar(42) NOT NULL," +
		"`num_actions` int(6) unsigned NOT NULL DEFAULT '0'," +
		"`timestamp` int(11) unsigned NOT NULL," +
		"PRIMARY KEY (`block_height`) USING BTREE," +
		"KEY `producer_address` (`producer_address`) USING BTREE" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return errors.Wrap(err, "failed to start block plugin")
	}

	return nil
}

func (b blockPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHash := blk.HashBlock()

	insertData := map[string]interface{}{
		"block_height":     blk.Height(),
		"block_hash":       hex.EncodeToString(blkHash[:]),
		"producer_address": blk.ProducerAddress(),
		"timestamp":        blk.Timestamp().Unix(),
	}

	err := kernel.Transaction(func(tx *sql.Tx) error {
		if err := kernel.InsertTableData(tx, b.tableName, insertData); err != nil {
			return err
		}
		return kernel.UpdateIndexHeight(tx, b.Name(), blk.Height())
	})
	return err
}

func (b blockPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockPlugin{
	tableName: "block",
}
