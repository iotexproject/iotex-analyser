package main

import (
	"encoding/hex"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/shopspring/decimal"
)

var specialActionHash = hash.ZeroHash256

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

func getActionType(t iotextypes.TransactionLogType) string {
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

func handleTransactionLogs(transactionLogs []*action.TransactionLog, actionHash string, blkHeight uint64) ([]models.BlockReceiptTransaction, error) {
	var brts []models.BlockReceiptTransaction
	for _, transation := range transactionLogs {
		transation := transation
		amountDec := decimal.NewFromBigInt(transation.Amount, 0)
		recipient := transation.Recipient
		if len(recipient) > 0 {
			if addr, err := address.FromString(recipient); err != nil {
				//skip invalid address
				continue
			} else {
				recipient = addr.String()
			}
		}
		brts = append(brts, models.BlockReceiptTransaction{
			BlockHeight: blkHeight,
			ActionHash:  actionHash,
			Type:        getActionType(transation.Type),
			Amount:      amountDec,
			Sender:      transation.Sender,
			Recipient:   recipient,
		})
	}

	return brts, nil
}

func handleLogs(logs []*action.Log, actionHash string, blkHeight uint64) ([]models.BlockReceiptLog, error) {
	var brls []models.BlockReceiptLog
	for _, log := range logs {
		log := log
		topic0, topic1, topic2, topic3 := parseTopics(log.Topics)
		logData := log.Data
		if logData == nil {
			logData = []byte("")
		}
		brls = append(brls, models.BlockReceiptLog{
			BlockHeight:        blkHeight,
			ActionHash:         actionHash,
			Address:            log.Address,
			Topic0:             topic0,
			Topic1:             topic1,
			Topic2:             topic2,
			Topic3:             topic3,
			Data:               logData,
			Index:              uint(log.Index),
			TxIndex:            uint(log.TxIndex),
			NotFixTopicCopyBug: log.NotFixTopicCopyBug,
		})
	}
	return brls, nil
}
