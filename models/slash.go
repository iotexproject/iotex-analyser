package models

import "github.com/shopspring/decimal"

type Slash struct {
	BlockHeight     uint64          `gorm:"not null;unsigned;index;uniqueIndex:idx_block_operator" sql:"type:bigint"`
	ActionHash      string          `gorm:"size:64;not null;index:,length:9"`
	OperatorAddress string          `gorm:"size:42;index;not null;uniqueIndex:idx_block_operator"`
	Amount          decimal.Decimal `gorm:"type:decimal(60,0);not null"`
}

func (Slash) TableName() string {
	return "slash"
}
