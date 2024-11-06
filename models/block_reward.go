package models

import "github.com/shopspring/decimal"

type AccountReward struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"not null;unsigned;index" sql:"type:bigint"`
	EpochNumber     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	CandidateName   string          `gorm:"size:42;index;not null"`
	BlockReward     decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	EpochReward     decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	FoundationBonus decimal.Decimal `gorm:"type:decimal(60,0);not null"`
}

func (AccountReward) TableName() string {
	return "account_rewards"
}

type BlockReward struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"not null;unsigned;index" sql:"type:bigint"`
	EpochNumber     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	RewardAddress   string          `gorm:"size:42;index;not null"`
	ActionHash      string          `gorm:"size:64;not null;index:,length:9"`
	CandidateName   string          `gorm:"size:42;index;not null"`
	BlockReward     decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	EpochReward     decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	FoundationBonus decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	PriorityBonus   decimal.Decimal `gorm:"type:decimal(60,0)"`
}

func (BlockReward) TableName() string {
	return "block_rewards"
}
