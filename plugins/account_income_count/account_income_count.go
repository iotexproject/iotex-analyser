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

const VERSION = "1.1.1"

type income struct {
	inFlow        *big.Int
	outFlow       *big.Int
	inNumActions  int
	outNumActions int
}

type accountIncomeCountPlugin struct {
	tableName string
}

func (b accountIncomeCountPlugin) Name() string {
	return "account_income_count"
}

func (b accountIncomeCountPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b accountIncomeCountPlugin) Start(ctx context.Context) error {
	createSql := "CREATE TABLE IF NOT EXISTS `" + b.tableName + "` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`account_address` varchar(41) NOT NULL DEFAULT ''," +
		"`in_flow` decimal(42,0) unsigned NOT NULL DEFAULT '0'," +
		"`in_num_actions` int(5) unsigned NOT NULL DEFAULT '0'," +
		"`out_flow` decimal(42,0) unsigned NOT NULL DEFAULT '0'," +
		"`out_num_actions` int(5) unsigned NOT NULL DEFAULT '0'," +
		"PRIMARY KEY (`id`)," +
		"UNIQUE KEY `account_address` (`account_address`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	config, _ := kernel.GetConfigCtx(ctx)
	_, err := newConfig(config)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s plugin config", b.Name())
	}

	query := "SELECT count(1) FROM `" + b.tableName + "`"
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
				"in_flow":         amount,
				"account_address": addr,
			}
			if err := kernel.InsertTableData(tx, b.tableName, insertData); err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}

func (b accountIncomeCountPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := kernel.Transaction(func(tx *sql.Tx) error {

		incomes := make(map[string]*income)
		for _, receipt := range blk.Receipts {

			//transaction
			for _, transation := range receipt.TransactionLogs() {
				if transation.Sender != "" {
					if _, ok := incomes[transation.Sender]; !ok {
						incomes[transation.Sender] = &income{
							outFlow:       big.NewInt(0).Set(transation.Amount),
							outNumActions: 1,
							inFlow:        big.NewInt(0),
							inNumActions:  0,
						}
					} else {
						incomes[transation.Sender].outFlow = incomes[transation.Sender].outFlow.Add(incomes[transation.Sender].outFlow, transation.Amount)
						incomes[transation.Sender].outNumActions += 1
					}
				}
				if transation.Recipient != "" {
					if _, ok := incomes[transation.Recipient]; !ok {
						incomes[transation.Recipient] = &income{
							inFlow:        big.NewInt(0).Set(transation.Amount),
							inNumActions:  1,
							outFlow:       big.NewInt(0),
							outNumActions: 0,
						}
					} else {
						incomes[transation.Recipient].inFlow = incomes[transation.Recipient].inFlow.Add(incomes[transation.Recipient].inFlow, transation.Amount)
						incomes[transation.Recipient].inNumActions += 1
					}
				}

			}
		}

		for accountAddress, income := range incomes {

			// timeStart := time.Now()
			_, err := tx.Exec("INSERT INTO "+b.tableName+" (account_address,in_flow,in_num_actions,out_flow,out_num_actions) VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE in_flow=in_flow+?,in_num_actions=in_num_actions+?,out_flow=out_flow+?,out_num_actions=out_num_actions+?", accountAddress, income.inFlow.String(), income.inNumActions, income.outFlow.String(), income.outNumActions, income.inFlow.String(), income.inNumActions, income.outFlow.String(), income.outNumActions)
			// log.L().Info("insert",
			// 	zap.Duration("timeSpent", time.Since(timeStart)),
			// )
			if err != nil {
				return err
			}
		}
		return kernel.UpdateIndexHeight(tx, b.Name(), blk.Height())
	})
	return err
}

func (b accountIncomeCountPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b accountIncomeCountPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = accountIncomeCountPlugin{
	tableName: "account_income_count",
}
