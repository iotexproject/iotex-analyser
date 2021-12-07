package main

import (
	"encoding/hex"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
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
