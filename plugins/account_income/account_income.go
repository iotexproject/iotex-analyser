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

const VERSION = "1.1.5"

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

	createSql = "CREATE TABLE IF NOT EXISTS `account_income_count` (" +
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
			insertData = map[string]interface{}{
				"in_flow":         amount,
				"account_address": addr,
			}
			if err := kernel.InsertTableData(tx, "account_income_count", insertData); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (b accountIncomePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	incomes := make(map[string]income)
	for _, receipt := range blk.Receipts {
		//transaction
		for _, transation := range receipt.TransactionLogs() {
			if transation.Sender != "" {
				inTran, ok := incomes[transation.Sender]
				if !ok {
					inTran = income{
						outFlow:       big.NewInt(0).Set(transation.Amount),
						outNumActions: 1,
						inFlow:        big.NewInt(0),
						inNumActions:  0,
					}
				} else {
					inTran.outFlow = inTran.outFlow.Add(inTran.outFlow, transation.Amount)
					inTran.outNumActions += 1
				}
				incomes[transation.Sender] = inTran
			}
			if transation.Recipient != "" {
				inTran, ok := incomes[transation.Recipient]
				if !ok {
					inTran = income{
						inFlow:        big.NewInt(0).Set(transation.Amount),
						inNumActions:  1,
						outFlow:       big.NewInt(0),
						outNumActions: 0,
					}
				} else {
					inTran.inFlow = inTran.inFlow.Add(inTran.inFlow, transation.Amount)
					inTran.inNumActions += 1
				}
				incomes[transation.Recipient] = inTran
			}

		}
	}
	blkHeight := blk.Height()
	err := kernel.Transaction(func(tx *sql.Tx) error {
		for accountAddress, accountIncome := range incomes {
			insertData := map[string]interface{}{
				"block_height":    blkHeight,
				"account_address": accountAddress,
				"in_flow":         accountIncome.inFlow.String(),
				"in_num_actions":  accountIncome.inNumActions,
				"out_flow":        accountIncome.outFlow.String(),
				"out_num_actions": accountIncome.outNumActions,
			}
			if err := kernel.InsertTableData(tx, b.tableName, insertData); err != nil {
				return err
			}

			var in_flow, out_flow sql.NullString
			var in_num_actions, out_num_actions sql.NullInt32

			in := new(big.Int)
			out := new(big.Int)
			query := "SELECT in_flow,in_num_actions,out_flow,out_num_actions FROM account_income_count WHERE account_address=?"
			if err := tx.QueryRow(query, accountAddress).Scan(&in_flow, &in_num_actions, &out_flow, &out_num_actions); err != nil {
				if err != sql.ErrNoRows {
					return err
				}
				insertData1 := map[string]interface{}{
					"account_address": accountAddress,
					"in_flow":         accountIncome.inFlow.String(),
					"in_num_actions":  accountIncome.inNumActions,
					"out_flow":        accountIncome.outFlow.String(),
					"out_num_actions": accountIncome.outNumActions,
				}
				if err := kernel.InsertTableData(tx, "account_income_count", insertData1); err != nil {
					return err
				}
			} else {
				in.SetString(in_flow.String, 10)
				out.SetString(out_flow.String, 10)
				in = in.Add(in, accountIncome.inFlow)
				out = out.Add(out, accountIncome.outFlow)
				inNum := int(in_num_actions.Int32) + accountIncome.inNumActions
				outNum := int(out_num_actions.Int32) + accountIncome.outNumActions

				updateData := map[string]interface{}{
					"in_flow":         in.String(),
					"in_num_actions":  inNum,
					"out_flow":        out.String(),
					"out_num_actions": outNum,
				}
				whereMap := map[string]interface{}{
					"account_address": accountAddress,
				}
				if err := kernel.UpdateTableData(tx, "account_income_count", updateData, whereMap); err != nil {
					return err
				}
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
