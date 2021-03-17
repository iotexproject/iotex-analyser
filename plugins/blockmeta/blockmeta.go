package main

import (
	"context"
	"database/sql"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
)

const VERSION = "1.0.1"

type blockMetaPlugin struct {
	tableName string
}

func (b blockMetaPlugin) Name() string {
	return "blockmeta"
}
func (b blockMetaPlugin) Start(ctx context.Context) error {
	createSql := "CREATE TABLE IF NOT EXISTS `" + b.tableName + "` (" +
		"`block_height` bigint(20) NOT NULL," +
		"`gas_consumed` int(11) NOT NULL," +
		"`producer_name` varchar(42) NOT NULL DEFAULT ''," +
		"PRIMARY KEY (`block_height`) USING BTREE" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return errors.Wrap(err, "failed to start blockmeta plugin")
	}

	return nil
}

func (b blockMetaPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	var gasConsumed uint64
	for _, receipt := range blk.Receipts {
		gasConsumed += receipt.GasConsumed
	}

	insertData := map[string]interface{}{
		"block_height":  blk.Height(),
		"gas_consumed":  gasConsumed,
		"producer_name": "",
	}

	err := kernel.Transaction(func(tx *sql.Tx) error {
		if err := kernel.InsertTableData(tx, b.tableName, insertData); err != nil {
			return err
		}
		return kernel.UpdateIndexHeight(tx, b.Name(), blk.Height())
	})
	return err
}

func (b blockMetaPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockMetaPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockMetaPlugin{
	tableName: "block_meta",
}
