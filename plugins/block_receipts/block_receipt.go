package main

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
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

type blockReceiptPlugin struct {
	tipHeight uint64
	minHeight uint64
	brs       []models.BlockReceipt
	brts      []models.BlockReceiptTransaction
	brls      []models.BlockReceiptLog
}

func (b *blockReceiptPlugin) Name() string {
	return "block_receipts"
}

func (b *blockReceiptPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *blockReceiptPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.BlockReceipt{}, &models.BlockReceiptLog{}, &models.BlockReceiptTransaction{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	var count int64
	if err := db.DB().Model(&models.BlockReceiptTransaction{}).Where("block_height=0").Count(&count).Error; err != nil {
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
				"amount":       amount,
				"sender":       "",
				"recipient":    addr,
			}
			if err := tx.Model(&models.BlockReceiptTransaction{}).Create(insertData).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (b *blockReceiptPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	receipts := blk.Receipts
	for _, receipt := range receipts {
		receipt := receipt
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		b.brs = append(b.brs, models.BlockReceipt{
			BlockHeight:        blk.Height(),
			ActionHash:         actionHash,
			GasConsumed:        receipt.GasConsumed,
			ContractAddress:    receipt.ContractAddress,
			ExecutionRevertMsg: strings.ReplaceAll(receipt.ExecutionRevertMsg(), string([]byte{0x00}), "0x00"),
			Status:             receipt.Status,
		})
		brt, err := handleTransactionLogs(receipt.TransactionLogs(), actionHash, blk.Height())
		if err != nil {
			return err
		}
		b.brts = append(b.brts, brt...)
		brl, err := handleLogs(receipt.Logs(), actionHash, blk.Height())
		if err != nil {
			return err
		}
		b.brls = append(b.brls, brl...)
	}
	if b.minHeight == 0 {
		b.minHeight = blk.Height()
	}
	return nil
}

func (b *blockReceiptPlugin) commit() error {
	brs := b.brs
	brts := b.brts
	brls := b.brls
	tipHeight := b.tipHeight
	b.brs = nil
	b.brts = nil
	b.brls = nil
	b.minHeight = 0
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(brs) > 0 {
			if err := tx.Model(&models.BlockReceipt{}).CreateInBatches(brs, 200).Error; err != nil {
				return err
			}
		}
		if len(brts) > 0 {
			if err := tx.Model(&models.BlockReceiptTransaction{}).CreateInBatches(brts, 200).Error; err != nil {
				return err
			}
		}
		if len(brls) > 0 {
			if err := tx.Model(&models.BlockReceiptLog{}).CreateInBatches(brls, 200).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *blockReceiptPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *blockReceiptPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *blockReceiptPlugin) BatchSize() int {
	return 1000
}

func (b *blockReceiptPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *blockReceiptPlugin) CatchUpSafe() bool { return true }

func (b *blockReceiptPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockReceiptPlugin{}
