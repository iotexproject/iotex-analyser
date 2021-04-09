package main

import (
	"context"
	"database/sql"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
)

const VERSION = "1.0.1"

type actionExecutionPlugin struct {
	tableName string
}

func (b actionExecutionPlugin) Name() string {
	return "action_execution"
}

func (b actionExecutionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b actionExecutionPlugin) Start(ctx context.Context) error {
	createSql := "CREATE TABLE IF NOT EXISTS `" + b.tableName + "` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`block_height` bigint(20) UNSIGNED NOT NULL DEFAULT 0," +
		"`action_hash` varchar(64) NOT NULL DEFAULT ''," +
		"`contract` varchar(41) NOT NULL DEFAULT ''," +
		"`receipt_contract_address` varchar(41) NOT NULL DEFAULT ''," +
		"`data` blob," +
		"PRIMARY KEY (`id`)," +
		"KEY `block_height` (`block_height`)," +
		"KEY `action_hash` (`action_hash`(9))," +
		"KEY `receipt_contract_address` (`receipt_contract_address`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b actionExecutionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := kernel.Transaction(func(tx *sql.Tx) error {
		for _, selp := range blk.Actions {
			actionHash := selp.Hash()

			act := selp.Action()
			var contract string
			var data []byte
			switch a := act.(type) {
			case *action.Execution:
				contract = a.Contract()
				data = a.Data()
			default:
				continue
			}
			insertData := map[string]interface{}{
				"block_height": blk.Height(),
				"action_hash":  hex.EncodeToString(actionHash[:]),
				"contract":     contract,
				"data":         data,
			}
			for _, receipt := range blk.Receipts {
				if receipt.ActionHash == actionHash {
					insertData["receipt_contract_address"] = receipt.ContractAddress
					break
				}
			}

			if err := kernel.InsertTableData(tx, b.tableName, insertData); err != nil {
				return err
			}
		}
		return kernel.UpdateIndexHeight(tx, b.Name(), blk.Height())
	})

	return err
}

func (b actionExecutionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b actionExecutionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = actionExecutionPlugin{
	tableName: "action_execution",
}
