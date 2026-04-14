package main

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.5.0"

const (
	transfer                   = "transfer"
	execution                  = "execution"
	depositToRewardingFund     = "depositToRewardingFund"
	claimFromRewardingFund     = "claimFromRewardingFund"
	stakeCreate                = "stakeCreate"
	stakeWithdraw              = "stakeWithdraw"
	stakeAddDeposit            = "stakeAddDeposit"
	candidateRegisterFee       = "candidateRegisterFee"
	candidateRegisterSelfStake = "candidateRegisterSelfStake"
	gasFee                     = "gasFee"
)

type blockReceiptTransactionPlugin struct {
	tipHeight uint64
	minHeight uint64
	brts      []BlockReceiptTransaction
}

func (b *blockReceiptTransactionPlugin) Name() string {
	return "block_receipt_transaction"
}

func (b *blockReceiptTransactionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *blockReceiptTransactionPlugin) DependentPlugins() []string {
	return []string{"account_meta"}
}

func (b *blockReceiptTransactionPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &BlockReceiptTransaction{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	var count int64
	if err := db.DB().Model(&BlockReceiptTransaction{}).Where("block_height=0").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hashStr := hex.EncodeToString(specialActionHash[:])
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for addr, amount := range config.Default.Genesis.Account.InitBalanceMap {
			insertData := map[string]interface{}{
				"block_height": uint64(0),
				"action_hash":  hashStr,
				"type":         "genesis",
				"internal":     false,
				"amount":       amount,
				"sender":       "",
				"recipient":    addr,
			}
			if err := tx.Model(&BlockReceiptTransaction{}).Create(insertData).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (b *blockReceiptTransactionPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	receipts := blk.Receipts
	for _, receipt := range receipts {
		receipt := receipt
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		brt, err := handleTransactionLogs(receipt.TransactionLogs(), actionHash, blk.Height())
		if err != nil {
			return err
		}
		b.brts = append(b.brts, brt...)
	}
	if b.minHeight == 0 {
		b.minHeight = blk.Height()
	}
	return nil
}

func (b *blockReceiptTransactionPlugin) commit() error {
	brts := b.brts
	tipHeight := b.tipHeight
	b.brts = nil
	b.minHeight = 0
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(brts) > 0 {
			if err := tx.Model(&BlockReceiptTransaction{}).CreateInBatches(brts, 200).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *blockReceiptTransactionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *blockReceiptTransactionPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *blockReceiptTransactionPlugin) BatchSize() int {
	return 1000
}

func (b *blockReceiptTransactionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *blockReceiptTransactionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = &blockReceiptTransactionPlugin{}
