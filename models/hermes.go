package models

import (
	"github.com/shopspring/decimal"
)

type HermesDistribute struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	StartEpoch      uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	EndEpoch        uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	DelegateName    string          `gorm:"size:42;index;not null"`
	NumOfRecipients uint64          `gorm:"unsigned"`
	TotalAmount     decimal.Decimal `gorm:"type:decimal(60,0);not null"`
}

func (HermesDistribute) TableName() string {
	return "hermes_distributes"
}

type HermesVotingResult struct {
	ID                        uint64          `gorm:"primary_key;" sql:"type:bigint"`
	EpochNumber               uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	DelegateName              string          `gorm:"size:42;index;not null"`
	OperatorAddress           string          `gorm:"size:42;index;not null"`
	RewardAddress             string          `gorm:"size:42;index;not null"`
	StakingAddress            string          `gorm:"size:42;index;not null"`
	TotalWeightedVotes        decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	SelfStaking               decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	BlockRewardPercentage     decimal.Decimal `gorm:"type:decimal(5,2);not null"`
	EpochRewardPercentage     decimal.Decimal `gorm:"type:decimal(5,2);not null"`
	FoundationBonusPercentage decimal.Decimal `gorm:"type:decimal(5,2);not null"`
}

func (HermesVotingResult) TableName() string {
	return "hermes_voting_results"
}

type HermesAggregateVoting struct {
	ID             uint64          `gorm:"primary_key;" sql:"type:bigint"`
	EpochNumber    uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	CandidateName  string          `gorm:"size:42;index;not null"`
	VoterAddress   string          `gorm:"size:42;index;not null"`
	NativeFlag     bool            `gorm:"type:bool;not null;default:false"`
	AggregateVotes decimal.Decimal `gorm:"type:decimal(60,0);not null"`
}

func (HermesAggregateVoting) TableName() string {
	return "hermes_aggregate_votings"
}

type HermesAccountReward struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	EpochNumber     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	CandidateName   string          `gorm:"size:42;index;not null"`
	BlockReward     decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	EpochReward     decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	FoundationBonus decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
}

func (HermesAccountReward) TableName() string {
	return "hermes_account_reward"
}
