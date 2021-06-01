package main

import "github.com/shopspring/decimal"

type BlockMeta struct {
	BlockHeight     uint64          `gorm:"primary_key;" sql:"type:bigint;"`
	GasConsumed     uint64          `gorm:"type:int4;unsigned;not null;default:0"`
	ProducerName    string          `gorm:"size:42;not null;default:'';"`
	ProducerAddress string          `gorm:"size:42;not null;default:'';"`
	BlockReward     decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
}

func (BlockMeta) TableName() string {
	return "block_meta"
}
