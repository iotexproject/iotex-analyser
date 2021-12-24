package models

import "github.com/shopspring/decimal"

type RewardHistory struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"not null;unsigned;index" sql:"type:bigint"`
	EpochNumber     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	RewardAddress   string          `gorm:"size:42;index;not null"`
	ActionHash      string          `gorm:"size:64;not null;index:,length:9"`
	CandidateName   string          `gorm:"size:42;index;not null"`
	BlockReward     decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	EpochReward     decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	FoundationBonus decimal.Decimal `gorm:"type:decimal(60,0);not null"`
}

func (RewardHistory) TableName() string {
	return "reward_history"
}
