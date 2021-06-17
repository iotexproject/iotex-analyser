package main

import (
	"github.com/shopspring/decimal"
)

type StakingBucket struct {
	ID         uint64          `gorm:"primary_key;" sql:"type:bigint"`
	ActionHash string          `gorm:"size:64;not null;index:,length:9"`
	BucketID   decimal.Decimal `gorm:"type:decimal(42,0);not null;index;default:0;"`
}

func (StakingBucket) TableName() string {
	return "staking_bucket"
}
