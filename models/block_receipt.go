package models

import "github.com/shopspring/decimal"

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
	return "block_receipts"
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
	return "block_receipt_transactions"
}

type BlockReceiptLog struct {
	ID                 uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight        uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash         string `gorm:"size:64;not null;index:,length:9"`
	Address            string `gorm:"size:64;not null;default:'';index:,length:9"`
	Topic0             string `gorm:"size:64;not null;default:'';index:,length:9"`
	Topic1             string `gorm:"size:64;not null;default:'';index:,length:9"`
	Topic2             string `gorm:"size:64;not null;default:'';index:,length:9"`
	Topic3             string `gorm:"size:64;not null;default:'';index:,length:9"`
	Data               []byte `gorm:"not null;"`
	Index              uint   `gorm:"type:int2;unsigned;not null;default:0"`
	TxIndex            uint   `gorm:"type:int2;unsigned;not null;default:0"`
	LogIndex           uint   `gorm:"type:int2;unsigned;not null;default:0"`
	NotFixTopicCopyBug bool   `gorm:"type:bool;not null;default:false"`
}

func (BlockReceiptLog) TableName() string {
	return "block_receipt_logs"
}
