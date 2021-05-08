package main

import (
	"context"
	"database/sql"
	"math/big"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
)

const VERSION = "1.0.1"

type income struct {
	inFlow        *big.Int
	outFlow       *big.Int
	inNumActions  int
	outNumActions int
}
type accountIncomePlugin struct {
	tableName string
}

func (b accountIncomePlugin) Name() string {
	return "account_income"
}

func (b accountIncomePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b accountIncomePlugin) Start(ctx context.Context) error {
	createSql := "CREATE TABLE IF NOT EXISTS `account_income` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`block_height` bigint(20) unsigned NOT NULL," +
		"`account_address` char(41) NOT NULL DEFAULT ''," +
		"`in_flow` decimal(42,0) unsigned NOT NULL DEFAULT '0'," +
		"`in_num_actions` int(5) unsigned NOT NULL DEFAULT '0'," +
		"`out_flow` decimal(42,0) unsigned NOT NULL DEFAULT '0'," +
		"`out_num_actions` int(5) unsigned NOT NULL DEFAULT '0'," +
		"PRIMARY KEY (`id`)," +
		"KEY `block_height` (`block_height`)," +
		"KEY `account_address` (`account_address`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return errors.Wrap(err, "failed to start plugin")
	}
	var err error
	config, _ := kernel.GetConfigCtx(ctx)
	_, err = newConfig(config)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s plugin config", b.Name())
	}

	query := "SELECT count(1) FROM `" + b.tableName + "` WHERE block_height=0"
	var count int
	if err := kernel.GetDB().QueryRow(query).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	err = kernel.Transaction(func(tx *sql.Tx) error {
		for addr, amount := range Default.Genesis.Account.InitBalanceMap {
			insertData := map[string]interface{}{
				"block_height":    uint64(0),
				"in_flow":         amount,
				"account_address": addr,
			}
			if err := kernel.InsertTableData(tx, b.tableName, insertData); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (b accountIncomePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := kernel.Transaction(func(tx *sql.Tx) error {

		incomes := make(map[string]*income)
		for _, receipt := range blk.Receipts {

			//transaction
			for _, transation := range receipt.TransactionLogs() {
				if transation.Sender != "" {
					if _, ok := incomes[transation.Sender]; !ok {
						incomes[transation.Sender] = &income{
							outFlow:       transation.Amount,
							outNumActions: 1,
							inFlow:        big.NewInt(0),
							inNumActions:  0,
						}
					} else {
						incomes[transation.Sender].outFlow.Add(incomes[transation.Sender].outFlow, transation.Amount)
						incomes[transation.Sender].outNumActions += 1
					}

				}
				if transation.Recipient != "" {
					if _, ok := incomes[transation.Recipient]; !ok {
						incomes[transation.Recipient] = &income{
							inFlow:        transation.Amount,
							inNumActions:  1,
							outFlow:       big.NewInt(0),
							outNumActions: 0,
						}
					} else {
						incomes[transation.Recipient].inFlow.Add(incomes[transation.Recipient].inFlow, transation.Amount)
						incomes[transation.Recipient].inNumActions += 1
					}

				}

			}
		}

		for accountAddress, income := range incomes {
			insertData := map[string]interface{}{
				"block_height":    blk.Height(),
				"account_address": accountAddress,
				"in_flow":         income.inFlow.String(),
				"in_num_actions":  income.inNumActions,
				"out_flow":        income.outFlow.String(),
				"out_num_actions": income.outNumActions,
			}
			if err := kernel.InsertTableData(tx, b.tableName, insertData); err != nil {
				return err
			}
		}
		return kernel.UpdateIndexHeight(tx, b.Name(), blk.Height())
	})

	return err
}

func (b accountIncomePlugin) Stop(ctx context.Context) error {
	return nil
}

func (b accountIncomePlugin) Version() string {
	return VERSION
}

// exported
var Plugin = accountIncomePlugin{
	tableName: "account_income",
}
