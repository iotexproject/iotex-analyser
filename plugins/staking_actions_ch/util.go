package main

import (
	"database/sql"

	"github.com/shopspring/decimal"

	"github.com/iotexproject/iotex-analyser/models"
)

func getCandidateAddressByName(name string, height uint64) (string, error) {
	candidate := &models.Candidate{}
	if err := candidate.FetchByNameWithHeight(name, height); err != nil {
		return "", err
	}
	return candidate.OwnerAddress, nil
}

func (b stakingActionChPlugin) getBucketSumAmountFromCacheByBucketID(bucketID uint64) (decimal.Decimal, error) {
	var amount sql.NullString
	zero := decimal.NewFromInt(0)
	if err := chDB.Model(&StakingActions{}).Select("sum(toDecimal256(amount, 10))").Where("bucket_id=?", bucketID).Scan(&amount).Error; err != nil {
		return zero, err
	}

	// sum from cache
	for _, action := range b.stakingActions {
		if action.BucketID == bucketID {
			zero = decimal.Sum(zero, action.Amount)
		}
	}

	if amount.String == "" {
		return zero, nil
	}
	decmailAmount, err := decimal.NewFromString(amount.String)
	if err != nil {
		return zero, err
	}
	return decimal.Sum(zero, decmailAmount), nil
}

func (b stakingActionChPlugin) getFixBucketSumAmountFromCacheByBucketID(bucketID uint64) (decimal.Decimal, error) {
	var amount sql.NullString
	var count int64
	zero := decimal.NewFromInt(0)

	err := chDB.Model(&StakingActions{}).Where("bucket_id=? and act_type='Unstake'", bucketID).Count(&count).Error
	if err != nil {
		return zero, err
	}

	for _, action := range b.stakingActions {
		if action.BucketID == bucketID && action.ActType == "Unstake" {
			count++
		}
	}

	if count == 0 {
		return zero, nil
	}
	if err := chDB.Model(&StakingActions{}).Select("sum(toDecimal256(amount, 10))").Where("bucket_id=? and act_type<>'Unstake'", bucketID).Scan(&amount).Error; err != nil {
		return zero, err
	}

	// sum from cache
	for _, action := range b.stakingActions {
		if action.BucketID == bucketID && action.ActType != "Unstake" {
			zero = decimal.Sum(zero, action.Amount)
		}
	}

	if amount.String == "" {
		return zero, nil
	}
	decmailAmount, err := decimal.NewFromString(amount.String)
	if err != nil {
		return zero, err
	}
	return decimal.Sum(zero, decmailAmount), nil
}

type BucketInfo struct {
	OwnerAddress string
	Candidate    string
	AutoStake    bool
	Duration     uint32
}

func (b stakingActionChPlugin) getBucketInfoAddressFromCacheByBucketID(bucketID uint64) (*BucketInfo, error) {
	var bi BucketInfo

	for i := len(b.stakingActions) - 1; i >= 0; i-- {
		if b.stakingActions[i].BucketID == bucketID {
			bi.OwnerAddress = b.stakingActions[i].OwnerAddress
			bi.Candidate = b.stakingActions[i].Candidate
			bi.AutoStake = b.stakingActions[i].AutoStake
			bi.Duration = b.stakingActions[i].Duration
			return &bi, nil
		}
	}

	if err := chDB.Model(&StakingActions{}).Select("owner_address,candidate,auto_stake,duration").Where("bucket_id=? and toDecimal256(amount, 10) > 0", bucketID).Order("block_height desc, index desc").Limit(1).Scan(&bi).Error; err != nil {
		return nil, err
	}
	return &bi, nil
}
