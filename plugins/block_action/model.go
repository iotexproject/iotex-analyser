package main

import (
	"github.com/shopspring/decimal"
)

type BlockAction struct {
	ID          uint64          `gorm:"primary_key;" sql:"type:bigint"`
	ActionHash  string          `gorm:"size:64;not null;index:,length:9"`
	ActionType  string          `gorm:"size:32;not null;index"`
	BlockHeight uint64          `gorm:"unsigned" sql:"type:bigint;index"`
	From        string          `gorm:"size:42;not null;default:'';index:,length:9"`
	To          string          `gorm:"size:42;not null;default:'';index:,length:9"`
	GasPrice    decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	GasLimit    uint64          `gorm:"type:int4;unsigned;not null;default:0"`
	Nonce       uint64          `gorm:"type:int8;unsigned;not null;default:0"`
	Amount      decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
}

func (BlockAction) TableName() string {
	return "block_action"
}
