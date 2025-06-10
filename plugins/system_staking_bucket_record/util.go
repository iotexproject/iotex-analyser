package main

import (
	"database/sql"
	"math/big"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/action/protocol/staking"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func getBucketSumAmountByBucketID(tx *gorm.DB, bucketID uint64) (decimal.Decimal, error) {
	var amount sql.NullString
	zero := decimal.NewFromInt(0)
	if err := tx.Model(&models.SystemStakingBucketRecord{}).Select("sum(amount)").Where("bucket_id=?", bucketID).Scan(&amount).Error; err != nil {
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
	VotingPower          string
	AutoStake            bool
	Duration             uint32
	DurationType         uint8
	CreateTime           int64
	StakeStartTime       int64
	UnstakeStartTime     int64
}

func getBucketInfoAddressByBucketID(tx *gorm.DB, bucketID uint64) (*BucketInfo, error) {
	var bi BucketInfo
	if err := tx.Model(&models.SystemStakingBucketRecord{}).Select("owner_address,delegate_owner_address,staked_amount,voting_power,auto_stake,duration,duration_type,create_time,stake_start_time,unstake_start_time").Where("bucket_id=?", bucketID).Last(&bi).Error; err != nil {
		return nil, err
	}
	return &bi, nil
}

func getVoteWeight(blkHeight uint64, duration time.Duration, stakeAmount *big.Int, autoStake, selfStake bool) *big.Int {
	if blkHeight < config.Default.Genesis.RedseaBlockHeight {
		return stakeAmount
	}
	voteBucket := &staking.VoteBucket{
		StakedAmount:   stakeAmount,
		AutoStake:      autoStake,
		StakedDuration: duration,
	}
	return staking.CalculateVoteWeight(config.Default.Genesis.Staking.VoteWeightCalConsts, voteBucket, selfStake)
}
