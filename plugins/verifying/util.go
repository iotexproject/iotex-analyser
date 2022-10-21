package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"unicode"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

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

var (
	// ErrActionNoMatch means action no match
	ErrActionNoMatch = errors.New("action no match")
	// ErrReceiptNoMatch means receipt no match
	ErrReceiptNoMatch = errors.New("receipt no match")
	// ErrTransactionNoMatch means transaction no match
	ErrTransactionNoMatch = errors.New("transaction no match")
)

func verifyAction(blk *block.Block, db *gorm.DB, height uint64) error {
	var block models.Block
	if err := db.Table("block").Where("block_height = ?", height).Find(&block).Error; err != nil {
		return err
	}
	blkHash := blk.HashBlock()
	if block.BlockHash != hex.EncodeToString(blkHash[:]) ||
		block.ProducerAddress != blk.ProducerAddress() ||
		block.Timestamp.Unix() != blk.Timestamp().Unix() ||
		block.NumActions != len(blk.Actions) {
		return ErrActionNoMatch
	}

	var actions []models.BlockAction
	if err := db.Table("block_action").Where("block_height = ?", height).Find(&actions).Error; err != nil {
		return err
	}
	if len(actions) != len(blk.Actions) {
		return errors.Wrapf(ErrActionNoMatch, "blkNum=%d action len not match", height)
	}
	for _, selp := range blk.Actions {
		actionHash, _ := selp.Hash()
		gasPrice := decimal.NewFromBigInt(selp.GasPrice(), 0)
		gasLimit := selp.GasLimit()
		nonce := selp.Nonce()

		act := selp.Action()
		actionType := getActionTypeString(act)
		amount, payload := getPayloadAmount(act)
		sender, _ := address.FromBytes(selp.SrcPubkey().Hash())
		dst, _ := selp.Destination()

		for i, action := range actions {
			if hex.EncodeToString(actionHash[:]) == action.ActionHash &&
				gasPrice.Equal(action.GasPrice) &&
				gasLimit == action.GasLimit &&
				nonce == action.Nonce &&
				actionType == action.ActionType &&
				action.Amount.Equal(decimal.NewFromBigInt(amount, 0)) &&
				bytes.Equal(payload, action.Payload) &&
				sender.String() == action.Sender &&
				dst == action.Recipient {
				actions = append(actions[:i], actions[i+1:]...)
				break
			}
		}
		if len(actions) == 0 {
			break
		}
	}
	if len(actions) != 0 {
		return errors.Wrapf(ErrActionNoMatch, "blkNum=%d action not match", height)
	}
	return nil
}

func verifyReceipt(blk *block.Block, db *gorm.DB, height uint64) error {
	daoReceipts := blk.Receipts
	var receipts []models.BlockReceipt
	if err := db.Table("block_receipts").Where("block_height = ?", height).Find(&receipts).Error; err != nil {
		return err
	}
	if len(daoReceipts) != len(receipts) {
		return errors.Wrapf(ErrReceiptNoMatch, "blkNum=%d receipt len not match", height)
	}
	for _, receipt := range daoReceipts {
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		for i, dbReceipt := range receipts {
			if actionHash == dbReceipt.ActionHash &&
				receipt.Status == dbReceipt.Status &&
				receipt.GasConsumed == dbReceipt.GasConsumed &&
				receipt.ContractAddress == dbReceipt.ContractAddress &&
				receipt.ExecutionRevertMsg() == dbReceipt.ExecutionRevertMsg {
				receipts = append(receipts[:i], receipts[i+1:]...)
				break
			}
		}
		if len(receipts) == 0 {
			break
		}
	}
	if len(receipts) != 0 {
		return errors.Wrapf(ErrReceiptNoMatch, "blkNum=%d receipt not match", height)
	}

	var logs []models.BlockReceiptLog
	if err := db.Table("block_receipt_logs").Where("block_height = ?", height).Find(&logs).Error; err != nil {
		return err
	}
	for _, receipt := range daoReceipts {
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		for _, log := range receipt.Logs() {
			for i, dbLog := range logs {
				topic0, topic1, topic2, topic3 := parseTopics(log.Topics)
				if actionHash == dbLog.ActionHash &&
					log.Address == dbLog.Address &&
					topic0 == dbLog.Topic0 &&
					topic1 == dbLog.Topic1 &&
					topic2 == dbLog.Topic2 &&
					topic3 == dbLog.Topic3 &&
					bytes.Equal(log.Data, dbLog.Data) {
					logs = append(logs[:i], logs[i+1:]...)
					break
				}
			}
		}
	}
	if len(logs) != 0 {
		return errors.Wrapf(ErrReceiptNoMatch, "blkNum=%d receipt log not match", height)
	}
	return nil
}

func verifyTransactions(blk *block.Block, db *gorm.DB, height uint64) error {
	daoReceipts := blk.Receipts
	var transactions []models.BlockReceiptTransaction
	if err := db.Table("block_receipt_transactions").Where("block_height = ?", height).Find(&transactions).Error; err != nil {
		return err
	}
	daoLen := 0
	for _, receipt := range daoReceipts {
		daoLen += len(receipt.TransactionLogs())
	}
	if daoLen != len(transactions) {
		return errors.Wrapf(ErrTransactionNoMatch, "blkNum=%d transaction len not match, dao[%d]!=db[%d]", height, daoLen, len(transactions))
	}
	for _, receipt := range daoReceipts {
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		for _, transaction := range receipt.TransactionLogs() {
			actType := getTransactionType(transaction.Type)
			ok := false
			for i, dbTransaction := range transactions {
				if actionHash == dbTransaction.ActionHash &&
					transaction.Sender == dbTransaction.Sender &&
					transaction.Recipient == dbTransaction.Recipient &&
					dbTransaction.Amount.String() == transaction.Amount.String() &&
					actType == dbTransaction.Type {
					transactions = append(transactions[:i], transactions[i+1:]...)
					ok = true
					break
				}
			}
			if !ok {
				return errors.Wrapf(ErrTransactionNoMatch, "blkNum=%d transaction not match", height)
			}
		}
	}

	return nil
}

func firstLowerCase(s string) string {
	if len(s) == 0 {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func getActionTypeString(action action.Action) string {
	actionType := fmt.Sprintf("%T", action)
	return firstLowerCase(strings.TrimLeft(actionType, "*action."))
}

func getPayloadAmount(act action.Action) (*big.Int, []byte) {
	amount := big.NewInt(0)

	var payload []byte
	switch a := act.(type) {
	case *action.Transfer:
		amount = a.Amount()
		payload = a.Payload()
	case *action.Execution:
		amount = a.Amount()
	case *action.DepositToRewardingFund:
		amount = a.Amount()
	case *action.ClaimFromRewardingFund:
		amount = a.Amount()
	case *action.CreateStake:
		amount = a.Amount()
		payload = a.Payload()
	case *action.DepositToStake:
		amount = a.Amount()
		payload = a.Payload()
	case *action.CandidateRegister:
		amount = a.Amount()
		payload = a.Payload()
	}
	return amount, payload
}

func getTransactionType(t iotextypes.TransactionLogType) string {
	switch {
	case t == iotextypes.TransactionLogType_IN_CONTRACT_TRANSFER:
		return execution
	case t == iotextypes.TransactionLogType_WITHDRAW_BUCKET:
		return stakeWithdraw
	case t == iotextypes.TransactionLogType_CREATE_BUCKET:
		return stakeCreate
	case t == iotextypes.TransactionLogType_DEPOSIT_TO_BUCKET:
		return stakeAddDeposit
	case t == iotextypes.TransactionLogType_CLAIM_FROM_REWARDING_FUND:
		return claimFromRewardingFund
	case t == iotextypes.TransactionLogType_DEPOSIT_TO_REWARDING_FUND:
		return depositToRewardingFund
	case t == iotextypes.TransactionLogType_CANDIDATE_REGISTRATION_FEE:
		return candidateRegisterFee
	case t == iotextypes.TransactionLogType_CANDIDATE_SELF_STAKE:
		return candidateRegisterSelfStake
	case t == iotextypes.TransactionLogType_GAS_FEE:
		return gasFee
	case t == iotextypes.TransactionLogType_NATIVE_TRANSFER:
		return transfer
	}
	return ""
}

func parseTopics(topics []hash.Hash256) (topic0, topic1, topic2, topic3 string) {
	if len(topics) > 3 {
		topic3 = hex.EncodeToString(topics[3][:])
	}
	if len(topics) > 2 {
		topic2 = hex.EncodeToString(topics[2][:])
	}
	if len(topics) > 1 {
		topic1 = hex.EncodeToString(topics[1][:])
	}
	if len(topics) > 0 {
		topic0 = hex.EncodeToString(topics[0][:])
	}
	return
}
