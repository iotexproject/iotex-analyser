package main

import (
	"bytes"
	"context"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.0.0"

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

const (
	receiptTableName            = "block_receipt"
	receiptTransactionTableName = "block_receipt_transaction"
	receiptLogTableName         = "block_receipt_log"
)

type blockReceiptPlugin struct {
}

func (b blockReceiptPlugin) Name() string {
	return "block_receipt"
}

func (b blockReceiptPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockReceiptPlugin) Start(ctx context.Context) error {
	var err error
	config, _ := kernel.GetConfigCtx(ctx)
	_, err = newConfig(config)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s plugin config", b.Name())
	}
	if err := db.DB().AutoMigrate(&BlockReceipt{}, &BlockReceiptLog{}, &BlockReceiptTransaction{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	var count int64
	if err := db.DB().Model(&BlockReceipt{}).Where("block_height=0").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hashStr := hex.EncodeToString(specialActionHash[:])
	err = db.DB().Transaction(func(tx *gorm.DB) error {
		for addr, amount := range Default.Genesis.Account.InitBalanceMap {
			insertData := map[string]interface{}{
				"block_height": uint64(0),
				"action_hash":  hashStr,
				"type":         "genesis",
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

func (b blockReceiptPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	receipts := blk.Receipts
	err := db.DB().Transaction(func(tx *gorm.DB) error {

		for _, receipt := range receipts {
			receipt := receipt
			actionHash := hex.EncodeToString(receipt.ActionHash[:])
			br := &BlockReceipt{
				BlockHeight:        blk.Height(),
				ActionHash:         actionHash,
				GasConsumed:        receipt.GasConsumed,
				ContractAddress:    receipt.ContractAddress,
				ExecutionRevertMsg: receipt.ExecutionRevertMsg(),
				Status:             receipt.Status,
			}
			if err := tx.Create(br).Error; err != nil {
				return err
			}
			//transaction
			for _, transation := range receipt.TransactionLogs() {
				transation := transation
				amountDec := decimal.NewFromBigInt(transation.Amount, 0)
				brt := &BlockReceiptTransaction{
					BlockHeight: blk.Height(),
					ActionHash:  actionHash,
					Type:        getActionType(transation.Type),
					Amount:      amountDec,
					Sender:      transation.Sender,
					Recipient:   transation.Recipient,
				}
				if err := tx.Create(brt).Error; err != nil {
					return err
				}
			}
			//logs
			for _, log := range receipt.Logs() {
				log := log
				topics := [][]byte{}
				for _, topic := range log.Topics {
					topics = append(topics, topic[:])
				}

				brl := &BlockReceiptLog{
					BlockHeight:        blk.Height(),
					ActionHash:         actionHash,
					Address:            log.Address,
					Topics:             bytes.Join(topics, []byte("\n")),
					Data:               log.Data,
					Index:              log.Index,
					NotFixTopicCopyBug: log.NotFixTopicCopyBug,
				}
				if err := tx.Create(brl).Error; err != nil {
					return err
				}
			}
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
