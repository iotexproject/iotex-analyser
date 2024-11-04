package main

import (
	"encoding/hex"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/action"
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

func isContractAddress(addr string) bool {
	m := &models.AccountMeta{}
	if err := m.ByAddress(addr); err != nil {
		return false
	}
	return m.IsContract
}

func handleTransactionLogs(transactionLogs []*action.TransactionLog, actionHash string, blkHeight uint64) ([]BlockReceiptTransaction, error) {
	var brts []BlockReceiptTransaction
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
		brts = append(brts, BlockReceiptTransaction{
			BlockHeight: blkHeight,
			ActionHash:  actionHash,
			Type:        getActionType(transation.Type),
			Internal:    isContractAddress(transation.Sender),
			Amount:      amountDec,
			Sender:      transation.Sender,
			Recipient:   recipient,
		})
	}

	return brts, nil
}
