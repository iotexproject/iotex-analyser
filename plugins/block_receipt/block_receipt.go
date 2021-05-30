package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
)

const VERSION = "1.1.2"

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
	if err = createTables(); err != nil {
		return errors.Wrapf(err, "failed to start %s plugin", b.Name())
	}
	return nil
}

func (b blockReceiptPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	receipts := blk.Receipts
	err := kernel.Transaction(func(tx *sql.Tx) error {

		for _, receipt := range receipts {
			receipt := receipt
			actionHash := hex.EncodeToString(receipt.ActionHash[:])
			insertData := map[string]interface{}{
				"block_height":         blk.Height(),
				"action_hash":          actionHash,
				"gas_consumed":         receipt.GasConsumed,
				"contract_address":     receipt.ContractAddress,
				"execution_revert_msg": receipt.ExecutionRevertMsg(),
				"status":               receipt.Status,
			}
			if err := kernel.InsertTableData(tx, receiptTableName, insertData); err != nil {
				return err
			}
			//transaction
			for _, transation := range receipt.TransactionLogs() {
				transation := transation
				insertData := map[string]interface{}{
					"block_height": blk.Height(),
					"action_hash":  actionHash,
					"type":         getActionType(transation.Type),
					"amount":       transation.Amount.String(),
					"sender":       transation.Sender,
					"recipient":    transation.Recipient,
				}
				if err := kernel.InsertTableData(tx, receiptTransactionTableName, insertData); err != nil {
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
				insertData := map[string]interface{}{
					"block_height":           blk.Height(),
					"action_hash":            hex.EncodeToString(log.ActionHash[:]),
					"address":                log.Address,
					"topics":                 bytes.Join(topics, []byte("\n")),
					"data":                   log.Data,
					"index":                  log.Index,
					"not_fix_topic_copy_bug": log.NotFixTopicCopyBug,
				}
				if err := kernel.InsertTableData(tx, receiptLogTableName, insertData); err != nil {
					return err
				}
			}
		}

		return kernel.UpdateIndexHeight(tx, b.Name(), blk.Height())
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
