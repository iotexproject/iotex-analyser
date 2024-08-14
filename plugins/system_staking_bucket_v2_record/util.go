package main

import (
	"database/sql"
	"math"
	"math/big"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func durationDays(duration uint32) uint32 {
	return duration / 17280
}

func getBucketSumAmountByBucketID(tx *gorm.DB, bucketID uint64) (decimal.Decimal, error) {
	var amount sql.NullString
	zero := decimal.NewFromInt(0)
	if err := tx.Model(&models.SystemStakingBucketV2Record{}).Select("sum(amount)").Where("bucket_id=?", bucketID).Scan(&amount).Error; err != nil {
		return zero, err
	}
	if amount.String == "" {
		return zero, nil
	}
	decmailAmount, err := decimal.NewFromString(amount.String)
	if err != nil {
		return zero, err
	}
	return decmailAmount, nil
}

type BucketInfo struct {
	OwnerAddress         string
	DelegateOwnerAddress string
	StakedAmount         string
	Amount string
	VotingPower          string
	AutoStake            bool
	Duration             uint32
	CreateTime           int64
	StakeStartTime       int64
	UnstakeStartTime     int64
}

func getBucketInfoAddressByBucketID(tx *gorm.DB, bucketID uint64) (*BucketInfo, error) {
	var bi BucketInfo
	if err := tx.Model(&models.SystemStakingBucketV2Record{}).Select("owner_address,delegate_owner_address,staked_amount,amount,voting_power,auto_stake,duration,create_time,stake_start_time,unstake_start_time").Where("bucket_id=?", bucketID).Last(&bi).Error; err != nil {
		return nil, err
	}
	return &bi, nil
}

type VoteBucket struct {
	Index            uint64
	Candidate        string
	Owner            string
	StakedAmount     *big.Int
	StakedDuration   uint32
	CreateTime       time.Time
	StakeStartTime   time.Time
	UnstakeStartTime time.Time
	AutoStake        bool
}

func getVoteWeight(blkHeight uint64, duration uint32, stakeAmount *big.Int, autoStake, selfStake bool) *big.Int {
	if blkHeight < config.Default.Genesis.RedseaBlockHeight {
		return stakeAmount
	}
	voteBucket := &VoteBucket{
		StakedAmount:   stakeAmount,
		AutoStake:      autoStake,
		StakedDuration: duration,
	}
	return calculateVoteWeight(config.Default.Genesis.Staking.VoteWeightCalConsts, voteBucket, selfStake)
}

func calculateVoteWeight(c genesis.VoteWeightCalConsts, v *VoteBucket, selfStake bool) *big.Int {
	remainingTime := float64(v.StakedDuration * 86400)
	weight := float64(1)
	var m float64
	if v.AutoStake {
		m = c.AutoStake
	}
	if remainingTime > 0 {
		weight += math.Log(math.Ceil(remainingTime/86400)*(1+m)) / math.Log(c.DurationLg) / 100
	}
	if selfStake && v.AutoStake && v.StakedDuration >= 91 {
		// self-stake extra bonus requires enable auto-stake for at least 3 months
		weight *= c.SelfStake
	}

	amount := new(big.Float).SetInt(v.StakedAmount)
	weightedAmount, _ := amount.Mul(amount, big.NewFloat(weight)).Int(nil)
	return weightedAmount
}
