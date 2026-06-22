package main

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-analyser/plugins/block_receipts_transaction/model"
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
}

func (b blockReceiptTransactionPlugin) Name() string {
	return "block_receipt_transaction"
}

func (b blockReceiptTransactionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockReceiptTransactionPlugin) DependentPlugins() []string {
	return []string{"account_meta"}
}

func (b blockReceiptTransactionPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &model.BlockReceiptTransaction{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	var count int64
	if err := db.DB().Model(&model.BlockReceiptTransaction{}).Where("block_height=0").Count(&count).Error; err != nil {
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
			if err := tx.Model(&model.BlockReceiptTransaction{}).Create(insertData).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (b blockReceiptTransactionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	receipts := blk.Receipts
	var brts []model.BlockReceiptTransaction
	for _, receipt := range receipts {
		receipt := receipt
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		//transaction
		brt, err := handleTransactionLogs(receipt.TransactionLogs(), actionHash, blk.Height())
		if err != nil {
			return err
		}
		brts = append(brts, brt...)
	}
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("block_height = ?", blk.Height()).Delete(&model.BlockReceiptTransaction{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.BlockReceiptTransaction{}).CreateInBatches(brts, 200).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err
}

func (b blockReceiptTransactionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockReceiptTransactionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockReceiptTransactionPlugin{}
