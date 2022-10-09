package main

import (
	"context"
	"math/big"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.1.3"

type income struct {
	inFlow        *big.Int
	outFlow       *big.Int
	inNumActions  int
	outNumActions int
}
type accountIncomePlugin struct {
}

func (b accountIncomePlugin) Name() string {
	return "account_income"
}

func (b accountIncomePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b accountIncomePlugin) Start(ctx context.Context) error {
	var ai *AccountIncome
	var aic *AccountIncomeCount
	if err := db.AutoMigrate(b.Name(), &AccountIncome{}, &AccountIncomeCount{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	var err error
	var count int64
	if err := db.DB().Model(ai).Where("block_height=0").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	err = db.DB().Transaction(func(tx *gorm.DB) error {
		initBalances := make(map[string]string)
		for addr, amount := range config.Default.Genesis.Account.InitBalanceMap {
			initBalances[addr] = amount
		}
		initBalances["io0000000000000000000000rewardingprotocol"] = config.Default.Genesis.Rewarding.InitBalanceStr
		for addr, amount := range initBalances {

			insertData := map[string]interface{}{
				"block_height": uint64(0),
				"in_flow":      amount,
				"address":      addr,
			}
			if err := tx.Model(ai).Create(insertData).Error; err != nil {
				return err
			}

			if err := tx.Model(aic).Create(map[string]interface{}{
				"in_flow": amount,
				"address": addr,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func getIncomes(blk *block.Block) (map[string]income, error) {
	incomes := make(map[string]income)
	receipts := blk.Receipts
	for _, receipt := range receipts {
		receipt := receipt
		//transaction
		for _, transation := range receipt.TransactionLogs() {
			transation := transation
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
					inTran.outNumActions++
				}
				incomes[transation.Sender] = inTran
			}
			recipient := transation.Recipient
			if len(recipient) > 0 {
				if addr, err := address.FromString(recipient); err != nil {
					//skip invalid address
					continue
				} else {
					recipient = addr.String()
				}
			}
			if recipient != "" {
				inTran, ok := incomes[recipient]
				if !ok {
					inTran = income{
						inFlow:        big.NewInt(0).Set(transation.Amount),
						inNumActions:  1,
						outFlow:       big.NewInt(0),
						outNumActions: 0,
					}
				} else {
					inTran.inFlow = inTran.inFlow.Add(inTran.inFlow, transation.Amount)
					inTran.inNumActions++
				}
				incomes[recipient] = inTran
			}

		}
	}
	return incomes, nil
}
func (b accountIncomePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	incomes, err := getIncomes(blk)
	if err != nil {
		return err
	}
	blkHeight := blk.Height()
	err = db.DB().Transaction(func(tx *gorm.DB) error {
		for accountAddress, accountIncome := range incomes {
			inFlow := decimal.NewFromBigInt(accountIncome.inFlow, 0)
			outFlow := decimal.NewFromBigInt(accountIncome.outFlow, 0)
			aim := &AccountIncome{
				BlockHeight:   blkHeight,
				Address:       accountAddress,
				InFlow:        inFlow,
				InNumActions:  accountIncome.inNumActions,
				OutFlow:       outFlow,
				OutNumActions: accountIncome.outNumActions,
			}
			if err := tx.Create(aim).Error; err != nil {
				return err
			}
			aicm := &AccountIncomeCount{
				Address:       accountAddress,
				InFlow:        inFlow,
				InNumActions:  accountIncome.inNumActions,
				OutFlow:       outFlow,
				OutNumActions: accountIncome.outNumActions,
			}
			var aic *AccountIncomeCount
			if err := tx.Where("address = ?", accountAddress).First(&aic).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
				if err := tx.Create(aicm).Error; err != nil {
					return err
				}
			} else {
				aic.InFlow = aic.InFlow.Add(inFlow)
				aic.InNumActions += accountIncome.inNumActions
				aic.OutFlow = aic.OutFlow.Add(outFlow)
				aic.OutNumActions += accountIncome.outNumActions

				if err := tx.Save(aic).Error; err != nil {
					return err
				}
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blkHeight)
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
var Plugin = accountIncomePlugin{}
