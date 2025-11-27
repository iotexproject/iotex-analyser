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
	EndorsementExpireHeight uint64          `gorm:"type:numeric;not null;default:0"`
	Timestamp               int64           `gorm:"type:int4;unsigned;not null;default:0"`
	AutoStake               bool
	Duration                uint32
}

func (StakingBucket) TableName() string {
	return "staking_buckets"
}

type StakingBucketShadow struct {
	*StakingBucket
}

func (StakingBucketShadow) TableName() string {
	return "staking_buckets_shadow"
}

type (
	SystemStakingBucketRecordBase struct {
		ID                   uint64          `gorm:"primary_key;" sql:"type:bigint"`
		BlockHeight          uint64          `gorm:"unsigned;index" sql:"type:bigint"`
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
		Duration             uint32 // days or seconds depending on DurationType
		Final                bool   `gorm:"type:bool;not null;default:false"`
		Muted                bool   `gorm:"type:bool;not null;default:false"`
		DurationType         uint8  // 0: 5s per block in days; 1: 2.5s per block in seconds; 2: seconds in seconds
	}

	SystemStakingBucketRecord struct {
		SystemStakingBucketRecordBase
		// No additional fields, just for clarity
	}

	SystemStakingBucketV2Record struct {
		SystemStakingBucketRecordBase
		// This struct is for the V2 version of the staking bucket record
	}

	SystemStakingBucketV3Record struct {
		SystemStakingBucketRecordBase
		// This struct is for the V3 version of the staking bucket record
	}
)

func (SystemStakingBucketRecord) TableName() string {
	return "system_staking_buckets_record"
}

func (SystemStakingBucketV2Record) TableName() string {
	return "system_staking_buckets_v2_record"
}

func (SystemStakingBucketV3Record) TableName() string {
	return "system_staking_buckets_v3_record"
}
