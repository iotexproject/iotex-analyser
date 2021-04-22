package main

import (
	"context"
	"database/sql"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
)

const VERSION = "1.1.0"

type tokenPlugin struct {
	tableName string
}

func (b tokenPlugin) Name() string {
	return "token_erc20"
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) Start(ctx context.Context) error {
	createSql := "CREATE TABLE IF NOT EXISTS `" + b.tableName + "` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`block_height` bigint(20) unsigned NOT NULL DEFAULT '0'," +
		"`action_hash` varchar(64) NOT NULL DEFAULT ''," +
		"`contract_address` varchar(41) NOT NULL DEFAULT ''," +
		"`amount` DECIMAL(42, 0) UNSIGNED NOT NULL DEFAULT 0," +
		"`from` varchar(41) NOT NULL DEFAULT ''," +
		"`to` varchar(41) NOT NULL DEFAULT ''," +
		"PRIMARY KEY (`id`)," +
		"KEY `block_height` (`block_height`)," +
		"KEY `from` (`from`)," +
		"KEY `to` (`to`)," +
		"KEY `contract_address` (`contract_address`)," +
		"KEY `action_hash` (`action_hash`(9))" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b tokenPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := kernel.Transaction(func(tx *sql.Tx) error {

		for _, receipt := range blk.Receipts {
			if receipt.Status != successStatus {
				continue
			}
			actionHash := hex.EncodeToString(receipt.ActionHash[:])
			for _, log := range receipt.Logs() {
				data := hex.EncodeToString(log.Data)
				var topics string
				for _, t := range log.Topics {
					topics += hex.EncodeToString(t[:])
				}
				if !checkTopics(topics, data) {
					continue
				}

				from, to, amount, err := ParseContractData(topics, data)
				if err != nil {
					return errors.Wrap(err, "failed to parse contract data")
				}
				insertData := map[string]interface{}{
					"block_height":     blk.Height(),
					"action_hash":      actionHash,
					"amount":           amount,
					"contract_address": log.Address,
					"from":             from,
					"to":               to,
				}
				if err := kernel.InsertTableData(tx, b.tableName, insertData); err != nil {
					return errors.Wrap(err, "failed to insert table data")
				}
			}
		}

		return kernel.UpdateIndexHeight(tx, b.Name(), blk.Height())
	})

	return err
}

func (b tokenPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b tokenPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = tokenPlugin{
	tableName: "token_erc20",
}
