package model

import "github.com/shopspring/decimal"

type BlockReceiptTransaction struct {
	ID          uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash  string          `gorm:"size:64;not null;index:,length:9"`
	Type        string          `gorm:"size:32;not null;default:'';"`
	Internal    bool            // internal transaction, the sender is contract
	Amount      decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	Sender      string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Recipient   string          `gorm:"size:42;not null;default:'';index:,length:9"`
}

func (BlockReceiptTransaction) TableName() string {
	return "block_receipt_transactions_v2"
}
