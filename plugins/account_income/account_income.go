package main

import (
	"context"
	"math/big"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const VERSION = "2.2.0"

type income struct {
	inFlow        *big.Int
	outFlow       *big.Int
	inNumActions  int
	outNumActions int
}

type accountIncomePlugin struct {
	tipHeight             uint64
	accountIncome         []*AccountIncome
	accountIncomeCountMap map[string]*AccountIncomeCount
}

func (b *accountIncomePlugin) Name() string {
	return "account_income"
}

func (b *accountIncomePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *accountIncomePlugin) BatchSize() int {
	return 1000
}

func (b *accountIncomePlugin) Start(ctx context.Context) error {
	b.accountIncomeCountMap = make(map[string]*AccountIncomeCount)

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

func (b *accountIncomePlugin) putBlock(ctx context.Context, blk *block.Block) error {
	incomes, err := getIncomes(blk)
	if err != nil {
		return err
	}
	blkHeight := blk.Height()
	for accountAddress, accountIncome := range incomes {
		inFlow := decimal.NewFromBigInt(accountIncome.inFlow, 0)
		outFlow := decimal.NewFromBigInt(accountIncome.outFlow, 0)
		b.accountIncome = append(b.accountIncome, &AccountIncome{
			BlockHeight:   blkHeight,
			Address:       accountAddress,
			InFlow:        inFlow,
			InNumActions:  accountIncome.inNumActions,
			OutFlow:       outFlow,
			OutNumActions: accountIncome.outNumActions,
		})
		if existing, ok := b.accountIncomeCountMap[accountAddress]; ok {
			existing.InFlow = existing.InFlow.Add(inFlow)
			existing.InNumActions += accountIncome.inNumActions
			existing.OutFlow = existing.OutFlow.Add(outFlow)
			existing.OutNumActions += accountIncome.outNumActions
		} else {
			b.accountIncomeCountMap[accountAddress] = &AccountIncomeCount{
				Address:       accountAddress,
				InFlow:        inFlow,
				InNumActions:  accountIncome.inNumActions,
				OutFlow:       outFlow,
				OutNumActions: accountIncome.outNumActions,
			}
		}
	}
	return nil
}

func (b *accountIncomePlugin) commit() error {
	accountIncome := b.accountIncome
	b.accountIncome = nil
	accountIncomeCountMap := b.accountIncomeCountMap
	b.accountIncomeCountMap = make(map[string]*AccountIncomeCount)
	tipHeight := b.tipHeight

	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(accountIncome) > 0 {
			if err := tx.Model(&AccountIncome{}).CreateInBatches(accountIncome, 2000).Error; err != nil {
				return err
			}
		}
		if len(accountIncomeCountMap) > 0 {
			deltas := make([]*AccountIncomeCount, 0, len(accountIncomeCountMap))
			for _, v := range accountIncomeCountMap {
				deltas = append(deltas, v)
			}
			if err := tx.Model(&AccountIncomeCount{}).Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "address"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"in_flow":        gorm.Expr("account_income_count.in_flow + EXCLUDED.in_flow"),
					"in_num_actions": gorm.Expr("account_income_count.in_num_actions + EXCLUDED.in_num_actions"),
					"out_flow":       gorm.Expr("account_income_count.out_flow + EXCLUDED.out_flow"),
					"out_num_actions": gorm.Expr("account_income_count.out_num_actions + EXCLUDED.out_num_actions"),
				}),
			}).CreateInBatches(deltas, 2000).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *accountIncomePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *accountIncomePlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *accountIncomePlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *accountIncomePlugin) Version() string {
	return VERSION
}

// exported
var Plugin = accountIncomePlugin{}
