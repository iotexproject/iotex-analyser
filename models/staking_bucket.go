package models

import "github.com/shopspring/decimal"

type StakingBucket struct {
	ID                      uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BucketID                uint64          `gorm:"unsigned;index"`
	BlockHeight             uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	CreateTime              int64           `gorm:"type:int4;unsigned;not null;default:0"`
	StakeStartTime          int64           `gorm:"type:int4;unsigned;not null;default:0"`
	UnstakeStartTime        int64           `gorm:"type:int4;unsigned;not null;default:0"`
	StakedAmount            decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	VotingPower             decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	OwnerAddress            string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Candidate               string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Amount                  decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	ActType                 string          `gorm:"size:42;not null;default:'';index"`
	Sender                  string          `gorm:"size:42;not null;default:'';index:,length:9"`
	ActionHash              string          `gorm:"size:64;not null;index:,length:9"`
	EndorsementExpireHeight uint64          `gorm:"unsigned;not null;default:0" sql:"type:bigint"`
	Timestamp               int64           `gorm:"type:int4;unsigned;not null;default:0"`
	AutoStake               bool
	Duration                uint32
}

func (StakingBucket) TableName() string {
	return "staking_buckets"
}

type SystemStakingBucket struct {
	ID                   uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight          uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	ContractAddress      string          `gorm:"size:42;not null;default:'';index:,length:9"`
	BucketID             uint64          `gorm:"unsigned;index"`
	CreateTime           int64           `gorm:"type:int4;unsigned;not null;default:0"`
	StakeStartTime       int64           `gorm:"type:int4;unsigned;not null;default:0"`
	UnstakeStartTime     int64           `gorm:"type:int4;unsigned;not null;default:0"`
	StakedAmount         decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	VotingPower          decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	OwnerAddress         string          `gorm:"size:42;not null;default:'';index:,length:9"`
	DelegateOwnerAddress string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Amount               decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	EventType            string          `gorm:"size:42;not null;default:'';index"`
	Sender               string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Recipient            string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Timestamp            int64           `gorm:"type:int4;unsigned;not null;default:0"`
	ActHash              string
	AutoStake            bool
	Duration             uint32 //means block number
}

func (SystemStakingBucket) TableName() string {
	return "system_staking_buckets"
}
