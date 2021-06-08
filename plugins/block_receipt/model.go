package main

import (
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/shopspring/decimal"
)

type BlockReceipt struct {
	ID                 uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight        uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash         string `gorm:"size:64;not null;index:,length:9"`
	GasConsumed        uint64 `gorm:"type:int4;unsigned;not null;default:0"`
	ContractAddress    string `gorm:"size:42;not null;default:'';"`
	Status             uint64 `gorm:"type:int2;unsigned;not null;default:0"`
	ExecutionRevertMsg string `gorm:"size:255;not null;default:''"`
}

func (BlockReceipt) TableName() string {
	return "block_receipt"
}

type BlockReceiptTransaction struct {
	ID          uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash  string          `gorm:"size:64;not null;index:,length:9"`
	Type        string          `gorm:"size:32;not null;default:'';"`
	Amount      decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	Sender      string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Recipient   string          `gorm:"size:42;not null;default:'';index:,length:9"`
}

func (BlockReceiptTransaction) TableName() string {
	return "block_receipt_transaction"
}

type BlockReceiptTransaction2 struct {
	ID          uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash  string          `gorm:"size:64;not null;index:,length:9"`
	UniqueKey   string          `gorm:"size:128;not null;unique"`
	Type        string          `gorm:"size:32;not null;default:'';"`
	Amount      decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	Sender      string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Recipient   string          `gorm:"size:42;not null;default:'';index:,length:9"`
}

func (BlockReceiptTransaction2) TableName() string {
	return "block_receipt_transaction_2"
}

type BlockReceiptLog struct {
	ID                 uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight        uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash         string `gorm:"size:64;not null;index:,length:9"`
	Address            string `gorm:"size:42;not null;default:'';index:,length:9"`
	Topics             []byte `gorm:"not null;"`
	Data               []byte `gorm:"not null;"`
	Index              uint   `gorm:"type:int2;unsigned;not null;default:0"`
	NotFixTopicCopyBug bool   `gorm:"type:bool;not null;default:false"`
}

func (BlockReceiptLog) TableName() string {
	return "block_receipt_log"
}

var specialActionHash = hash.ZeroHash256

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
