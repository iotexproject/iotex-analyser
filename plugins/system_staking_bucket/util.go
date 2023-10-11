package main

import (
	"database/sql"
	"math/big"
	"time"

	"github.com/iotexproject/iotex-analyser/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func durationDays(duration uint32) uint32 {
	return duration / 17280
}

func getBucketSumAmountByBucketID(tx *gorm.DB, bucketID uint64) (decimal.Decimal, error) {
	var amount sql.NullString
	zero := decimal.NewFromInt(0)
	if err := tx.Model(&models.SystemStakingBucket{}).Select("sum(amount)").Where("bucket_id=?", bucketID).Scan(&amount).Error; err != nil {
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
	OwnerAddress     string
	Candidate        string
	StakedAmount     string
	VotingPower      string
	AutoStake        bool
	Duration         uint32
	CreateTime       int64
	StakeStartTime   int64
	UnstakeStartTime int64
}

func getBucketInfoAddressByBucketID(tx *gorm.DB, bucketID uint64) (*BucketInfo, error) {
	var bi BucketInfo
	if err := tx.Model(&models.SystemStakingBucket{}).Select("owner_address,candidate,staked_amount,voting_power,auto_stake,duration,create_time,stake_start_time,unstake_start_time").Where("bucket_id=?", bucketID).Last(&bi).Error; err != nil {
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
