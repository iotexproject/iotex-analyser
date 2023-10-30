package main

import "github.com/shopspring/decimal"

type HermesAggregateVoting struct {
	ID             uint64          `gorm:"primary_key;" sql:"type:bigint"`
	EpochNumber    uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	CandidateName  string          `gorm:"size:42;index;not null"`
	VoterAddress   string          `gorm:"size:42;index;not null"`
	NativeFlag     bool            `gorm:"type:bool;not null;default:false"`
	AggregateVotes decimal.Decimal `gorm:"type:decimal(60,0);not null"`
}

func (HermesAggregateVoting) TableName() string {
	return "hermes_aggregate_votings_fix"
}

type HermesVotingMeta struct {
	EpochNumber        uint64          `gorm:"unsigned;uniqueIndex" sql:"type:bigint"`
	VotedToken         decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	DelegateCount      int             `gorm:"type:int;not null;default:0;"`
	TotalWeightedVotes decimal.Decimal `gorm:"type:decimal(60,0);not null"`
}

func (HermesVotingMeta) TableName() string {
	return "hermes_voting_meta_fix"
}

type SystemStakingBucket struct {
	BucketID             uint64
	OwnerAddress         string
	DelegateOwnerAddress string
	StakedAmount         string
	AutoStake            bool
	Duration             uint32
}
