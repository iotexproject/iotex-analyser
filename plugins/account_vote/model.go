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
	if err := db.DB().Model(&models.Delegate{}).Select("operator_address").Where("name=?", name).Scan(&addr).Error; err != nil {
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

// getBucketSumAmountByBucketID returns the sum of all amount values for the given bucket_id.
func getBucketSumAmountByBucketID(tx *gorm.DB, bucketID uint64) (decimal.Decimal, error) {
	var amount sql.NullString
	zero := decimal.NewFromInt(0)
	if err := tx.Model(&AccountVote{}).Select("sum(amount)").Where("bucket_id=?", bucketID).Scan(&amount).Error; err != nil {
		return zero, err
	}
	if !amount.Valid || amount.String == "" {
		return zero, nil
	}
	d, err := decimal.NewFromString(amount.String)
	if err != nil {
		return zero, err
	}
	return d, nil
}

type BucketInfo struct {
	BucketID  uint64
	Address   string
	Candidate string
	AutoStake bool
	Duration  uint32
}

// getBucketInfoByBucketID returns the most-recent bucket info for the given bucket_id.
func getBucketInfoByBucketID(tx *gorm.DB, bucketID uint64) (*BucketInfo, error) {
	var bi BucketInfo
	if err := tx.Model(&AccountVote{}).
		Select("bucket_id, address, candidate, auto_stake, duration").
		Where("bucket_id=?", bucketID).Order("id desc").Limit(1).Scan(&bi).Error; err != nil {
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

// getBucketIDsByAddressWithHeight returns all bucket_ids whose most-recent record (at or before
// height) has address == addr. Replaces the former N+1-query loop with a single SQL round-trip.
func getBucketIDsByAddressWithHeight(tx *gorm.DB, addr string, height uint64) ([]uint64, error) {
	// Step 1: candidate buckets — any bucket_id where addr ever appeared up to height.
	candidateBuckets := tx.Table("account_vote").
		Select("bucket_id").
		Where("address = ? AND block_height <= ?", addr, height)

	// Step 2: for each candidate bucket find the row with the highest id (most recent) at or before height.
	maxIDSubQ := tx.Table("account_vote").
		Select("bucket_id, MAX(id) as max_id").
		Where("bucket_id IN (?) AND block_height <= ?", candidateBuckets, height).
		Group("bucket_id")

	// Step 3: keep only those whose most-recent row still points to addr.
	var results []struct {
		BucketID uint64 `gorm:"column:bucket_id"`
	}
	if err := tx.Table("account_vote AS t").
		Select("t.bucket_id").
		Joins("JOIN (?) AS sub ON t.id = sub.max_id", maxIDSubQ).
		Where("t.address = ?", addr).
		Scan(&results).Error; err != nil {
		return nil, err
	}
	bucketIDs := make([]uint64, 0, len(results))
	for _, r := range results {
		bucketIDs = append(bucketIDs, r.BucketID)
	}
	return bucketIDs, nil
}

func getForwardToAddressByFrom(tx *gorm.DB, addr string) (string, error) {
	var to string
	if err := tx.Model(&AccountVote{}).Select("forward_to").Where("address=? and act_type='GovernaceForward' and forward_to!=''", addr).Order("id desc").Limit(1).Scan(&to).Error; err != nil {
		return "", err
	}
	return to, nil
}
