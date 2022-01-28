package main

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestHermes(t *testing.T) {
	require := require.New(t)
	config.Default.Database = config.Database{
		Driver:   "postgres",
		Name:     "mainnet",
		Host:     "scout.cluster-cpx5likuxolf.us-west-1.rds.amazonaws.com",
		Port:     "5432",
		User:     "scout",
		Password: "ScoutPssw0rd",
		Debug:    true,
	}
	_, err := db.Connect()
	require.NoError(err)
	err = rebuildAccountRewardTable2(23830)
	require.NoError(err)
}

func rebuildAccountRewardTable2(lastEpoch uint64) error {
	if lastEpoch == 0 {
		return nil
	}
	db := db.DB()
	// Get voting result from last epoch
	rewardAddrToNameMapping, weightedVotesMapping, err := getVotingInfo(lastEpoch)
	if err != nil {
		if errors.Is(err, ErrEmptyRecords) {
			return nil
		}
		return errors.Wrap(err, "failed to get voting info")
	}
	// Get aggregate reward	records from last epoch
	var rows []AggregateReward
	if err := db.Raw("SELECT epoch_number, reward_address, SUM(block_reward) block_reward, SUM(epoch_reward)epoch_reward, SUM(foundation_bonus)foundation_bonus "+
		"FROM block_rewards WHERE epoch_number = ? GROUP BY epoch_number, reward_address", lastEpoch).Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		epochNumber := row.EpochNumber
		rewardAddress := row.RewardAddress
		candidateNames := rewardAddrToNameMapping[rewardAddress]
		// Multiple delegates share reward address
		totalBlockReward, ok := big.NewInt(0).SetString(row.BlockReward, 10)
		if !ok {
			return errors.New("failed to convert string to big int")
		}
		totalEpochReward, ok := big.NewInt(0).SetString(row.EpochReward, 10)
		if !ok {
			return errors.New("failed to convert string to big int")
		}
		totalFoundationBonus, ok := big.NewInt(0).SetString(row.FoundationBonus, 10)
		if !ok {
			return errors.New("failed to convert string to big int")
		}
		if len(candidateNames) == 1 {
			continue
		}
		candidateRewardsMap, err := breakdownRewards(epochNumber, candidateNames, weightedVotesMapping,
			totalBlockReward, totalEpochReward, totalFoundationBonus)
		if err != nil {
			return errors.Wrap(err, "failed to get candidate rewards map")
		}
		fmt.Printf("candidateRewardsMap = %d\n", len(candidateRewardsMap))
		for candidateName, rewards := range candidateRewardsMap {
			modelHAR := &models.HermesAccountReward{
				EpochNumber:     lastEpoch,
				CandidateName:   candidateName,
				BlockReward:     decimal.NewFromBigInt(rewards[0], 0),
				EpochReward:     decimal.NewFromBigInt(rewards[1], 0),
				FoundationBonus: decimal.NewFromBigInt(rewards[2], 0),
			}
			fmt.Printf("%v\n", modelHAR)
		}
	}
	return nil
}
