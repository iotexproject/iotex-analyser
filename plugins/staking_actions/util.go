package main

import (
	"database/sql"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func getCandidateAddressByName(name string, height uint64) (string, error) {
	candidate := &models.Candidate{}
	if err := candidate.FetchByNameWithHeight(name, height); err != nil {
		return "", err
	}
	return candidate.CandidateID, nil
}

func getBucketSumAmountByBucketID(tx *gorm.DB, bucketID uint64) (decimal.Decimal, error) {
	var amount sql.NullString
	zero := decimal.NewFromInt(0)
	if err := tx.Model(&models.StakingActions{}).Select("sum(amount)").Where("bucket_id=?", bucketID).Scan(&amount).Error; err != nil {
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

func getFixBucketSumAmountByBucketID(tx *gorm.DB, bucketID uint64) (decimal.Decimal, error) {
	var amount sql.NullString
	var count int64
	zero := decimal.NewFromInt(0)
	err := tx.Model(&models.StakingActions{}).Where("bucket_id=? and act_type='Unstake'", bucketID).Count(&count).Error
	if err != nil {
		return zero, err
	}
	if count == 0 {
		return zero, nil
	}
	if err := tx.Model(&models.StakingActions{}).Select("sum(amount)").Where("bucket_id=? and act_type<>'Unstake'", bucketID).Scan(&amount).Error; err != nil {
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
	OwnerAddress string
	Candidate    string
	AutoStake    bool
	Duration     uint32
}

func getBucketInfoAddressByBucketID(tx *gorm.DB, bucketID uint64) (*BucketInfo, error) {
	var bi BucketInfo
	if err := tx.Model(&models.StakingActions{}).Select("owner_address,candidate,auto_stake,duration").Where("bucket_id=?", bucketID).Order("id desc").Limit(1).Scan(&bi).Error; err != nil {
		return nil, err
	}
	return &bi, nil
}

func getAddresFromHash256(h hash.Hash256) (address.Address, error) {
	hexStr := hex.EncodeToString(h[:])
	ethAddr := hexStr[24:]
	ethAddress := common.HexToAddress(ethAddr)
	return address.FromBytes(ethAddress.Bytes())
}

func getBucketIDsByAddressWithHeight(addr string, height uint64) ([]uint64, error) {
	db := db.DB()
	var ids []struct {
		BucketID uint64
	}
	if err := db.Model(&models.StakingActions{}).Distinct("bucket_id").Where("block_height<=? and owner_address=?", height, addr).Find(&ids).Error; err != nil {
		return nil, err
	}
	bucketID := []uint64{}
	for _, id := range ids {
		bucketOwner, _ := getBucketOwnerWithHeight(id.BucketID, height)
		if addr != bucketOwner {
			continue
		}
		bucketID = append(bucketID, id.BucketID)
	}
	return bucketID, nil
}

func getBucketOwnerWithHeight(bucketID, height uint64) (string, error) {
	var addr sql.NullString
	db := db.DB()
	if err := db.Model(&models.StakingActions{}).Select("owner_address").Where("block_height<=? and bucket_id=?", height, bucketID).Order("id desc").Limit(1).Scan(&addr).Error; err != nil {
		return "", err
	}
	return addr.String, nil
}

func getForwardToAddressByFrom(addr string) (string, error) {
	var to string
	db := db.DB()
	if err := db.Model(&models.StakingActions{}).Select("forward_to").Where("owner_address=? and act_type='GovernaceForward' and forward_to!=''", addr).Order("id desc").Limit(1).Scan(&to).Error; err != nil {
		return "", err
	}
	return to, nil
}
