package main

import "github.com/shopspring/decimal"

type Candidate struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"not null;unsigned;index" sql:"type:bigint"`
	Name            string          `gorm:"size:42;not null;default:'';index"`
	OperatorAddress string          `gorm:"size:42;not null;default:'';"`
	RewardAddress   string          `gorm:"size:42;not null;default:'';"`
	OwnerAddress    string          `gorm:"size:42;not null;default:'';"`
	Amount          decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	ActType         string
	AutoStake       bool
	Duration        uint32          `gorm:"not null;" sql:"type:bigint"`
	Nonce           uint64          `gorm:"type:int8;unsigned;not null;default:0"`
	GasLimit        uint64          `gorm:"type:int4;unsigned;not null;default:0"`
	GasPrice        decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	Payload         []byte
}

func (Candidate) TableName() string {
	return "candidate"
}
