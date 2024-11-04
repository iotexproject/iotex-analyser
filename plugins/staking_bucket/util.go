package main

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const (
	protocolID       = "staking"
	readBucketsLimit = 300000
)

func GetStakingBucketByID(bucketID, blkHeight uint64) (*iotextypes.VoteBucket, error) {
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	chainClient := kernel.ChainClient()

	for i := uint32(0); ; i++ {
		offset := i * readBucketsLimit
		size := uint32(readBucketsLimit)
		voteBucketList, err := getStakingBuckets(chainClient, offset, size, epochHeight)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get bucket")
		}
		for _, bucket := range voteBucketList.Buckets {
			if bucket.Index == bucketID {
				return bucket, nil
			}
		}
		if len(voteBucketList.Buckets) < readBucketsLimit {
			break
		}
	}
	return nil, nil
}

// getStakingBuckets get specific buckets by height
func getStakingBuckets(chainClient iotexapi.APIServiceClient, offset, limit uint32, height uint64) (voteBucketList *iotextypes.VoteBucketList, err error) {
	methodName, err := proto.Marshal(&iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_BUCKETS,
	})
	if err != nil {
		return nil, err
	}
	arguments := &iotexapi.ReadStakingDataRequest{
		Request: &iotexapi.ReadStakingDataRequest_Buckets{
			Buckets: &iotexapi.ReadStakingDataRequest_VoteBuckets{
				Pagination: &iotexapi.PaginationParam{
					Offset: offset,
					Limit:  limit,
				},
			},
		},
	}
	argumentsBytes, _ := proto.Marshal(arguments)
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte(protocolID),
		MethodName: methodName,
		Arguments:  [][]byte{argumentsBytes},
		Height:     fmt.Sprintf("%d", height),
	}
	ctx := context.WithValue(context.Background(), &iotexapi.ReadStateRequest{}, iotexapi.ReadStakingDataMethod_BUCKETS)
	readStateRes, err := chainClient.ReadState(ctx, readStateRequest)
	if err != nil {
		return
	}
	voteBucketList = &iotextypes.VoteBucketList{}
	if err := proto.Unmarshal(readStateRes.GetData(), voteBucketList); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal VoteBucketList")
	}
	return
}

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
	if err := tx.Model(&models.StakingBucket{}).Select("sum(amount)").Where("bucket_id=?", bucketID).Scan(&amount).Error; err != nil {
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
	err := tx.Model(&models.StakingBucket{}).Where("bucket_id=? and act_type='Unstake'", bucketID).Count(&count).Error
	if err != nil {
		return zero, err
	}
	if count == 0 {
		return zero, nil
	}
	if err := tx.Model(&models.StakingBucket{}).Select("sum(amount)").Where("bucket_id=? and act_type<>'Unstake'", bucketID).Scan(&amount).Error; err != nil {
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
	AutoStake        bool
	Duration         uint32
	CreateTime       int64
	StakeStartTime   int64
	UnstakeStartTime int64
}

func getBucketInfoAddressByBucketID(tx *gorm.DB, bucketID uint64) (*BucketInfo, error) {
	var bi BucketInfo
	if err := tx.Model(&models.StakingBucket{}).Select("owner_address,candidate,auto_stake,duration,create_time,stake_start_time,unstake_start_time").Where("bucket_id=?", bucketID).Last(&bi).Error; err != nil {
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

func getVoteWeight(duration uint32, stakeAmount *big.Int, autoStake, selfStake bool) *big.Int {
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
