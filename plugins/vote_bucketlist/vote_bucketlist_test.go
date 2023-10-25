package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestVoteBucketList(t *testing.T) {
	require := require.New(t)
	_, err := initTestConfig()
	require.NoError(err)
	epochNum := uint64(20220)
	chainClient := kernel.ChainClient()
	list1, err := models.GetVoteBucketList(epochNum)
	require.NoError(err)
	epochHeight := kernel.GetEpochHeight(epochNum)
	list2, err := GetAllStakingBuckets(chainClient, epochHeight)
	require.NoError(err)
	require.Equal(len(list1.Buckets), len(list2.Buckets))
	for _, bucket := range list1.Buckets {
		found := false
		for _, bucket2 := range list2.Buckets {
			if bucket.GetStakedAmount() == bucket2.GetStakedAmount() &&
				bucket.GetAutoStake() == bucket2.GetAutoStake() &&
				bucket.GetStakedDuration() == bucket2.GetStakedDuration() &&
				bucket.GetOwner() == bucket2.GetOwner() &&
				bucket.GetCreateTime().AsTime().Unix() == bucket2.GetCreateTime().AsTime().Unix() &&
				bucket.GetIndex() == bucket2.GetIndex() &&
				bucket.GetUnstakeStartTime().AsTime().Unix() == bucket2.GetUnstakeStartTime().AsTime().Unix() {
				found = true
				break
			}
		}
		require.True(found, fmt.Sprintf("bucket %d not found", bucket.Index))
	}
}

func TestRange(t *testing.T) {
	require := require.New(t)
	_, err := initTestConfig()
	require.NoError(err)
	startEpoch := uint64(22352)
	endEpoch := uint64(22354)
	for i := startEpoch; i <= endEpoch; i++ {
		epochNum := i
		list1, err := models.GetVoteBucketList(epochNum)
		require.NoError(err)
		epochHeight := kernel.GetEpochHeight(epochNum)
		chainClient := kernel.ChainClient()
		list2, err := GetAllStakingBuckets(chainClient, epochHeight)
		require.NoError(err)
		require.Equal(len(list1.Buckets), len(list2.Buckets))
		for i, bucket := range list1.Buckets {
			require.Equal(bucket.GetStakedAmount(), list2.Buckets[i].GetStakedAmount())
			require.Equal(bucket.GetAutoStake(), list2.Buckets[i].GetAutoStake())
			require.Equal(bucket.GetStakedDuration(), list2.Buckets[i].GetStakedDuration())
			require.Equal(bucket.GetOwner(), list2.Buckets[i].GetOwner())
			require.Equal(bucket.GetCreateTime(), list2.Buckets[i].GetCreateTime())
			require.Equal(bucket.GetIndex(), list2.Buckets[i].GetIndex())
			require.Equal(bucket.GetUnstakeStartTime(), list2.Buckets[i].GetUnstakeStartTime())
		}
		fmt.Printf("epoch: %d passed\n", epochNum)
	}
}

func initTestConfig() (*gorm.DB, error) {
	_, err := config.New(os.Getenv("ConfigPath"))
	if err != nil {
		return nil, err
	}
	return db.Connect()
}
