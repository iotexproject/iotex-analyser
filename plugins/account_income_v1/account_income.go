package main

import (
	"context"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-analyser/server"
	"github.com/iotexproject/iotex-core/blockchain/block"
	slog "github.com/iotexproject/iotex-core/pkg/log"
)

const VERSION = "2.1.3"

type income struct {
	inFlow        *big.Int
	outFlow       *big.Int
	inNumActions  int
	outNumActions int
}
type accountIncomeV1Plugin struct {
	batchSize             int
	tipHeight             uint64
	accountIncome         []*AccountIncomeV1
	accountIncomeCountMap map[string]*AccountIncomeCountV1
}

func (b accountIncomeV1Plugin) Name() string {
	return "account_income_v1"
}

func (b accountIncomeV1Plugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b accountIncomeV1Plugin) BatchSize() int {
	return b.batchSize
}

func (b *accountIncomeV1Plugin) Start(ctx context.Context) error {
	b.accountIncomeCountMap = make(map[string]*AccountIncomeCountV1)

	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		cfg := &Config{}
		if err := yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			b.batchSize = cfg.BatchSize
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}

	var ai *AccountIncomeV1
	var aic *AccountIncomeCountV1
	if err := db.AutoMigrate(b.Name(), &AccountIncomeV1{}, &AccountIncomeCountV1{}); err != nil {
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

func (b *accountIncomeV1Plugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	startTime := float64(time.Now().UnixNano()) / 1e9
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}

	server.OpDurationMtc.WithLabelValues("account_income_v1", "putBlocks").Set(float64(time.Now().UnixNano())/1e9 - startTime)
	b.tipHeight = blks[0].Height() + uint64(len(blks)) - 1
	err := b.commit()
	return err
}

func (b accountIncomeV1Plugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *accountIncomeV1Plugin) putBlock(ctx context.Context, blk *block.Block) error {
	incomes, err := getIncomes(blk)
	if err != nil {
		return err
	}
	blkHeight := blk.Height()
	for accountAddress, accountIncome := range incomes {
		inFlow := decimal.NewFromBigInt(accountIncome.inFlow, 0)
		outFlow := decimal.NewFromBigInt(accountIncome.outFlow, 0)
		b.accountIncome = append(b.accountIncome, &AccountIncomeV1{
			BlockHeight:   blkHeight,
			Address:       accountAddress,
			InFlow:        inFlow,
			InNumActions:  accountIncome.inNumActions,
			OutFlow:       outFlow,
			OutNumActions: accountIncome.outNumActions,
		})
		aicm := &AccountIncomeCountV1{
			Address:       accountAddress,
			InFlow:        inFlow,
			InNumActions:  accountIncome.inNumActions,
			OutFlow:       outFlow,
			OutNumActions: accountIncome.outNumActions,
		}
		if err := b.appendToAccountIncomeCount(aicm); err != nil {
			return err
		}
	}

	return nil
}

func (b *accountIncomeV1Plugin) commit() error {
	startTime := float64(time.Now().UnixNano()) / 1e9
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if len(b.accountIncome) > 0 {
			if err := db.DB().Model(&AccountIncomeV1{}).CreateInBatches(b.accountIncome, 2000).Error; err != nil {
				slog.L().Error("put accountIncome ", zap.String("plugin", b.Name()), zap.Int("size", len(b.accountIncome)))
				b.accountIncome = b.accountIncome[:0]
				return err
			}
			b.accountIncome = b.accountIncome[:0]
		}
		if len(b.accountIncomeCountMap) > 0 {
			accountIncomeCounts := make([]*AccountIncomeCountV1, 0, len(b.accountIncomeCountMap))
			for _, v := range b.accountIncomeCountMap {
				accountIncomeCounts = append(accountIncomeCounts, v)
			}
			if err := db.DB().Model(&AccountIncomeCountV1{}).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"address", "in_flow", "in_num_actions", "out_flow", "out_num_actions"}),
			}).CreateInBatches(accountIncomeCounts, 2000).Error; err != nil {
				slog.L().Error("put accountIncomeCounts ", zap.String("plugin", b.Name()), zap.Int("size", len(accountIncomeCounts)))
				b.accountIncomeCountMap = make(map[string]*AccountIncomeCountV1)
				return err
			}
			b.accountIncomeCountMap = make(map[string]*AccountIncomeCountV1)
		}

		return db.UpdateIndexHeightByTx(tx, b.Name(), b.tipHeight)
	})
	server.OpDurationMtc.WithLabelValues("account_income_v1", "commit").Set(float64(time.Now().UnixNano())/1e9 - startTime)
	return err
}

func (b accountIncomeV1Plugin) appendToAccountIncomeCount(aicm *AccountIncomeCountV1) error {
	var aic *AccountIncomeCountV1
	accountIncomeCount, ok := b.accountIncomeCountMap[aicm.Address]
	if !ok {
		if err := db.DB().Where("address = ?", aicm.Address).First(&aic).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			b.accountIncomeCountMap[aicm.Address] = aicm
			return nil
		}
		b.accountIncomeCountMap[aicm.Address] = aic
		return nil
	}
	accountIncomeCount.InFlow = accountIncomeCount.InFlow.Add(aicm.InFlow)
	accountIncomeCount.InNumActions += aicm.InNumActions
	accountIncomeCount.OutFlow = accountIncomeCount.OutFlow.Add(aicm.OutFlow)
	accountIncomeCount.OutNumActions += aicm.OutNumActions

	return nil
}

func (b accountIncomeV1Plugin) Stop(ctx context.Context) error {
	return nil
}

func (b accountIncomeV1Plugin) Version() string {
	return VERSION
}

// exported
var Plugin = accountIncomeV1Plugin{}
