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

func TestCandidateList(t *testing.T) {
	require := require.New(t)
	_, err := initTestConfig()
	require.NoError(err)
	epochNum := uint64(20221)
	chainClient := kernel.ChainClient()
	list1, err := models.GetCandidateList(epochNum)
	require.NoError(err)
	epochHeight := kernel.GetEpochHeight(epochNum)
	list2, err := GetAllStakingCandidates(chainClient, epochHeight)
	require.NoError(err)
	require.Equal(len(list1.GetCandidates()), len(list2.GetCandidates()))
	for i, bucket := range list1.GetCandidates() {
		require.Equal(bucket.GetName(), list2.Candidates[i].GetName())
		require.Equal(bucket.GetOperatorAddress(), list2.Candidates[i].GetOperatorAddress())
		require.Equal(bucket.GetOwnerAddress(), list2.Candidates[i].GetOwnerAddress())
		require.Equal(bucket.GetRewardAddress(), list2.Candidates[i].GetRewardAddress())
		require.Equal(bucket.GetSelfStakingTokens(), list2.Candidates[i].GetSelfStakingTokens())
		require.Equal(bucket.GetTotalWeightedVotes(), list2.Candidates[i].GetTotalWeightedVotes())
		require.Equal(bucket.GetSelfStakeBucketIdx(), list2.Candidates[i].GetSelfStakeBucketIdx())
	}
}

/*
   expected: "3226575558987914492587314"
   actual  : "3226576024740550463520062"
*/

func TestRange(t *testing.T) {
	require := require.New(t)
	_, err := initTestConfig()
	require.NoError(err)
	startEpoch := uint64(22352)
	endEpoch := uint64(22354)
	for i := startEpoch; i <= endEpoch; i++ {
		epochNum := i
		chainClient := kernel.ChainClient()
		list1, err := models.GetCandidateList(epochNum)
		require.NoError(err)
		epochHeight := kernel.GetEpochHeight(epochNum)
		list2, err := GetAllStakingCandidates(chainClient, epochHeight)
		require.NoError(err)
		require.Equal(len(list1.GetCandidates()), len(list2.GetCandidates()))
		for i, bucket := range list1.GetCandidates() {
			require.Equal(bucket.GetName(), list2.Candidates[i].GetName())
			require.Equal(bucket.GetOperatorAddress(), list2.Candidates[i].GetOperatorAddress())
			require.Equal(bucket.GetOwnerAddress(), list2.Candidates[i].GetOwnerAddress())
			require.Equal(bucket.GetRewardAddress(), list2.Candidates[i].GetRewardAddress())
			require.Equal(bucket.GetSelfStakingTokens(), list2.Candidates[i].GetSelfStakingTokens())
			require.Equal(bucket.GetTotalWeightedVotes(), list2.Candidates[i].GetTotalWeightedVotes())
			require.Equal(bucket.GetSelfStakeBucketIdx(), list2.Candidates[i].GetSelfStakeBucketIdx())
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
