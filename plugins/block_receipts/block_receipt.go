package main

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
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
}

func (b blockReceiptPlugin) Name() string {
	return "block_receipts"
}

func (b blockReceiptPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockReceiptPlugin) Start(ctx context.Context) error {
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

func (b blockReceiptPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	receipts := blk.Receipts
	var brs []models.BlockReceipt
	var brts []models.BlockReceiptTransaction
	var brls []models.BlockReceiptLog
	for _, receipt := range receipts {
		receipt := receipt
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		brs = append(brs, models.BlockReceipt{
			BlockHeight:        blk.Height(),
			ActionHash:         actionHash,
			GasConsumed:        receipt.GasConsumed,
			ContractAddress:    receipt.ContractAddress,
			ExecutionRevertMsg: strings.ReplaceAll(receipt.ExecutionRevertMsg(), string([]byte{0x00}), "0x00"),
			Status:             receipt.Status,
		})
		//transaction
		brt, err := handleTransactionLogs(receipt.TransactionLogs(), actionHash, blk.Height())
		if err != nil {
			return err
		}
		brts = append(brts, brt...)
		//logs
		brl, err := handleLogs(receipt.Logs(), actionHash, blk.Height())
		if err != nil {
			return err
		}
		brls = append(brls, brl...)
	}
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("block_height = ?", blk.Height()).Delete(&models.BlockReceipt{}).Error; err != nil {
			return err
		}
		if err := tx.Where("block_height = ?", blk.Height()).Delete(&models.BlockReceiptTransaction{}).Error; err != nil {
			return err
		}
		if err := tx.Where("block_height = ?", blk.Height()).Delete(&models.BlockReceiptLog{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.BlockReceipt{}).CreateInBatches(brs, 200).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.BlockReceiptTransaction{}).CreateInBatches(brts, 200).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.BlockReceiptLog{}).CreateInBatches(brls, 200).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err
}

func (b blockReceiptPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockReceiptPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockReceiptPlugin{}
