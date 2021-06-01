package main

import (
	"github.com/shopspring/decimal"
)

type TokenErc721 struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"unsigned" sql:"type:bigint;index"`
	ActionHash      string          `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string          `gorm:"size:42;not null;default:'';index,length:9"`
	Amount          decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	From            string          `gorm:"size:42;not null;default:'';index:,length:9"`
	To              string          `gorm:"size:42;not null;default:'';index:,length:9"`
}

func (TokenErc721) TableName() string {
	return "token_erc721"
}
