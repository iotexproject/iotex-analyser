package main

import (
	"github.com/shopspring/decimal"
)

type AccountIncome struct {
	ID            uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight   uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	Address       string          `gorm:"size:42;not null;default:'';index:,length:9"`
	InFlow        decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	InNumActions  int             `gorm:"type:int;unsigned;not null;default:0"`
	OutFlow       decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	OutNumActions int             `gorm:"type:int;unsigned;not null;default:0"`
}

func (AccountIncome) TableName() string {
	return "account_income"
}

type AccountIncomeCount struct {
	ID            uint64          `gorm:"primary_key;" sql:"type:bigint"`
	Address       string          `gorm:"size:42;not null;default:'';uniqueIndex"`
	InFlow        decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	InNumActions  int             `gorm:"type:int;unsigned;not null;default:0"`
	OutFlow       decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	OutNumActions int             `gorm:"type:int;unsigned;not null;default:0"`
}

func (AccountIncomeCount) TableName() string {
	return "account_income_count"
}
