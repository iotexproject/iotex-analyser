package main

import (
	"database/sql"
	"encoding/json"
	"math"
	"math/big"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
)

const (
	StoreKey   = "airdrip"
	StepHeight = uint64(100)
)

type storeJson struct {
	CurrentHeight uint64    `json:"currentHeight"`
	NextHeight    uint64    `json:"nextHeight"`
	UpdateTime    time.Time `json:"UpdateTime"`
}

func getAirdripFromStore() (*storeJson, error) {
	var res storeJson
	store := &db.Store{
		Key: StoreKey,
	}
	if err := db.DB().Where(store).First(store).Error; err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(store.Value), &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func calcNextHeight(height uint64) uint64 {
	nextHeight := uint64(0)
	if height >= Default.Airdrip.InitHeight {
		nextHeight = Default.Airdrip.InitHeight + ((height-Default.Airdrip.InitHeight)/StepHeight+1)*StepHeight
	} else {
		nextHeight = Default.Airdrip.InitHeight
	}
	return nextHeight
}

func getAliveBucketIDs(height uint64) ([]uint64, error) {
	db := db.DB()
	var ids []struct {
		BucketID uint64
	}
	height1 := uint64(0)
	if height > StepHeight {
		height1 = height - StepHeight
	}
	height2 := uint64(0)
	if height > 90*StepHeight {
		height2 = height - 90*StepHeight
	}
	query := "select bucket_id from staking_action where bucket_id in (select distinct on (bucket_id) bucket_id from staking_action where block_height >=? and block_height<=? and auto_stake order by bucket_id,id desc) and block_height<=?"
	if err := db.Raw(query, height1, height, height2).Find(&ids).Error; err != nil {
		return nil, err
	}
	bucketID := []uint64{}
	for _, id := range ids {
		bucketID = append(bucketID, id.BucketID)
	}
	return bucketID, nil
}

type OwnerVote struct {
	StakeAmount *big.Int
	VoteWeight  *big.Int
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

type AccountVote struct {
	ID          uint64
	BlockHeight uint64
	BucketID    uint64
	Address     string
	Candidate   string
	Amount      string
	ActType     string
	AutoStake   bool
	Duration    uint32
}

func getBucketOwnerWithHeight(bucketID, height uint64) (string, error) {
	var addr sql.NullString
	db := db.DB()
	if err := db.Table("staing_action").Select("address").Where("block_height<=? and bucket_id=?", height, bucketID).Order("id desc").Limit(1).Scan(&addr).Error; err != nil {
		return "", err
	}
	return addr.String, nil
}

func getSumStake(addr string, height, bucketID uint64) (*big.Int, error) {
	db := db.DB()
	var amount sql.NullString
	height1 := uint64(0)
	if height > StepHeight {
		height1 = height - StepHeight
	}
	if err := db.Table("staking_action").Select("sum(amount)").Where("block_height >=? and block_height<=? and bucket_id=? and address=?", height1, height, bucketID, addr).Scan(&amount).Error; err != nil {
		return nil, err
	}
	stakeAmount, _ := big.NewInt(0).SetString(amount.String, 0)
	return stakeAmount, nil
}

func getVoteBucketParams(addr string, height, bucketID uint64) (uint32, bool, bool) {
	var av models.StakingAction
	db := db.DB()
	height1 := uint64(0)
	if height > StepHeight {
		height1 = height - StepHeight
	}
	if err := db.Table("staking_actions").Where("block_height >=? and block_height<=? and bucket_id=? and address=?", height1, height, bucketID, addr).Order("id desc").Scan(&av).Error; err != nil {
		return 0, false, false
	}

	selfAutoStake := false
	if addr == av.Candidate {
		selfAutoStake = true
	}
	return av.Duration, av.AutoStake, selfAutoStake
}
