package main

import (
	"database/sql"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AccountVote struct {
	ID          uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	BucketID    uint64          `gorm:"unsigned;index"`
	Address     string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Candidate   string          `gorm:"size:42;not null;default:'';index:,length:9"`
	ForwardTo   string          `gorm:"size:42;not null;default:'';"` //store Governace Forward To Address
	Amount      decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	ActType     string
	AutoStake   bool
	Duration    uint32
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
	if err := db.DB().Model(&models.StakingBucket{}).Select("bucket_id").Where("action_hash=?", actHash).Scan(&bucketID).Error; err != nil {
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

func getBucketSumAmountByBucketID(tx *gorm.DB, bucketID uint64) (decimal.Decimal, error) {
	var amount sql.NullString
	zero := decimal.NewFromInt(0)
	if err := tx.Model(&AccountVote{}).Select("sum(amount)").Where("bucket_id=?", bucketID).Scan(&amount).Error; err != nil {
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
	Address   string
	Candidate string
	AutoStake bool
	Duration  uint32
}

func getBucketInfoAddressByBucketID(tx *gorm.DB, bucketID uint64) (*BucketInfo, error) {
	var bi BucketInfo
	if err := tx.Model(&AccountVote{}).Select("address,candidate,auto_stake,duration").Where("bucket_id=?", bucketID).Order("id desc").Limit(1).Scan(&bi).Error; err != nil {
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
	if err := db.Table("account_vote").Distinct("bucket_id").Where("block_height<=? and address=?", height, addr).Find(&ids).Error; err != nil {
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
	if err := db.Model(&AccountVote{}).Select("address").Where("block_height<=? and bucket_id=?", height, bucketID).Order("id desc").Limit(1).Scan(&addr).Error; err != nil {
		return "", err
	}
	return addr.String, nil
}

func getForwardToAddressByFrom(addr string) (string, error) {
	var to string
	db := db.DB()
	if err := db.Model(&AccountVote{}).Select("forward_to").Where("address=? and act_type='GovernaceForward' and forward_to!=''", addr).Order("id desc").Limit(1).Scan(&to).Error; err != nil {
		return "", err
	}
	return to, nil
}
