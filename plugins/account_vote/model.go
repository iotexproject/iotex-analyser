package main

import (
	"math"
	"math/big"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	"github.com/shopspring/decimal"
)

type AccountVote struct {
	ID                    uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight           uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	BucketID              uint64          `gorm:"unsigned;index"`
	Address               string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Candidate             string          `gorm:"size:42;not null;default:'';index:,length:9"`
	CreateStakeAmount     decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	UnStakeAmount         decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	CreateStakeVoteWeight decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	UnStakeVoteWeight     decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
}

func (AccountVote) TableName() string {
	return "account_vote"
}

func getCandidateAddressByName(name string) (string, error) {
	var addr string
	if err := db.DB().Table("node_delegates").Select("producer_address").Where("producer_name=?", name).Scan(&addr).Error; err != nil {
		return "", err
	}
	return addr, nil
}

func getBucketIDByActHash(actHash string) (uint64, error) {
	var bucketID uint64
	if err := db.DB().Table("staking_bucket").Select("bucket_id").Where("action_hash=?", actHash).Scan(&bucketID).Error; err != nil {
		return 0, err
	}
	return bucketID, nil
}

type VoteBucket struct {
	Index            uint64
	Candidate        string
	Owner            string
	StakedAmount     *big.Int
	StakedDuration   time.Duration
	CreateTime       time.Time
	StakeStartTime   time.Time
	UnstakeStartTime time.Time
	AutoStake        bool
}

func calculateVoteWeight(c genesis.VoteWeightCalConsts, v *VoteBucket, selfStake bool) *big.Int {
	remainingTime := v.StakedDuration.Seconds()
	weight := float64(1)
	var m float64
	if v.AutoStake {
		m = c.AutoStake
	}
	if remainingTime > 0 {
		weight += math.Log(math.Ceil(remainingTime/86400)*(1+m)) / math.Log(c.DurationLg) / 100
	}
	if selfStake && v.AutoStake && v.StakedDuration >= time.Duration(91)*24*time.Hour {
		// self-stake extra bonus requires enable auto-stake for at least 3 months
		weight *= c.SelfStake
	}

	amount := new(big.Float).SetInt(v.StakedAmount)
	weightedAmount, _ := amount.Mul(amount, big.NewFloat(weight)).Int(nil)
	return weightedAmount
}
