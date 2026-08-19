package models

import "github.com/shopspring/decimal"

type RewardHistory struct {
	ID            uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight   uint64 `gorm:"not null;unsigned;index" sql:"type:bigint"`
	EpochNumber   uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	RewardAddress string `gorm:"size:42;index;not null"`
	ActionHash    string `gorm:"size:64;not null;index:,length:9"`
	CandidateName string `gorm:"size:42;index;not null"`
	// PayoutAddrKind is "reward" or "owner" — which candidate role
	// RewardAddress played when this payout was made. Since Zanzibar an
	// IIP-59 delegate is paid its commission at its owner address while a
	// legacy delegate is paid the full amount at its reward address, so this
	// is the cheap discriminator between a commission row and a full-reward
	// row. Empty for rows indexed before this column existed, and ambiguous
	// (always "reward") when a candidate uses one address for both roles.
	PayoutAddrKind  string          `gorm:"size:8;not null;default:''"`
	BlockReward     decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	EpochReward     decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	FoundationBonus decimal.Decimal `gorm:"type:decimal(60,0);not null"`
}

func (RewardHistory) TableName() string {
	return "reward_history"
}
