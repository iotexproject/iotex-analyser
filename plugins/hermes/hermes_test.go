package main

import (
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestHermes(t *testing.T) {
	require := require.New(t)
	// config.Default.Database = config.Database{
	// 	Driver:   "postgres",
	// 	Name:     "mainnet",
	// 	Host:     "scout.cluster-cpx5likuxolf.us-west-1.rds.amazonaws.com",
	// 	Port:     "5432",
	// 	User:     "scout",
	// 	Password: "ScoutPssw0rd",
	// 	Debug:    true,
	// }
	// config.Default.Database = config.Database{
	// 	Driver:   "postgres",
	// 	Name:     "mainnet",
	// 	Host:     "35.245.68.77",
	// 	Port:     "5432",
	// 	User:     "postgres",
	// 	Password: "MevbHde6PJN9F3zxrh61ViF9q3SeFKzD",
	// 	Debug:    true,
	// }
	config.Default.Database = config.Database{
		Driver:   "postgres",
		Name:     "mainlive",
		Host:     "127.0.0.1",
		Port:     "5432",
		User:     "postgres",
		Password: "admin",
		Debug:    true,
	}
	_, err := db.Connect()
	require.NoError(err)
	err = rebuildAccountRewardTable2(24738)
	require.NoError(err)
}

func rebuildAccountRewardTable2(lastEpoch uint64) error {
	if lastEpoch == 0 {
		return nil
	}
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		// Get voting result from last epoch
		rewardAddrToNameMapping, weightedVotesMapping, err := getVotingInfo(tx, lastEpoch)
		if err != nil {
			if errors.Is(err, ErrEmptyRecords) {
				return nil
			}
			return errors.Wrap(err, "failed to get voting info")
		}
		// Get aggregate reward	records from last epoch
		var rows []AggregateReward
		if err := tx.Raw("SELECT epoch_number, reward_address, SUM(block_reward) block_reward, SUM(epoch_reward)epoch_reward, SUM(foundation_bonus)foundation_bonus "+
			"FROM block_rewards WHERE epoch_number = ? GROUP BY epoch_number, reward_address", lastEpoch).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) > 0 {
			err := tx.Where("epoch_number = ?", lastEpoch).Delete(&models.HermesAccountReward{}).Error
			if err != nil {
				return err
			}
		}
		ii := 0
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
				candidateName := candidateNames[0]
				modelHAR := models.HermesAccountReward{
					EpochNumber:     lastEpoch,
					CandidateName:   candidateName,
					BlockReward:     decimal.NewFromBigInt(totalBlockReward, 0),
					EpochReward:     decimal.NewFromBigInt(totalEpochReward, 0),
					FoundationBonus: decimal.NewFromBigInt(totalFoundationBonus, 0),
				}
				fmt.Printf("%d %+v\n", ii, modelHAR)
				if err := tx.Create(&modelHAR).Error; err != nil {
					return err
				}
				ii++
				continue
			}
			candidateRewardsMap, err := breakdownRewards(tx, epochNumber, candidateNames, weightedVotesMapping,
				totalBlockReward, totalEpochReward, totalFoundationBonus)
			if err != nil {
				return errors.Wrap(err, "failed to get candidate rewards map")
			}
			fmt.Printf("candidateRewardsMap = %d\n", len(candidateRewardsMap))
			for candidateName, rewards := range candidateRewardsMap {
				modelHAR := models.HermesAccountReward{
					EpochNumber:     lastEpoch,
					CandidateName:   candidateName,
					BlockReward:     decimal.NewFromBigInt(rewards[0], 0),
					EpochReward:     decimal.NewFromBigInt(rewards[1], 0),
					FoundationBonus: decimal.NewFromBigInt(rewards[2], 0),
				}
				fmt.Printf("%d %+v\n", ii, modelHAR)
				ii++
				if err := tx.Create(&modelHAR).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	return err
}

func TestProbationList(t *testing.T) {
	require := require.New(t)
	_, err := initTestConfig()
	require.NoError(err)
	epochNum := uint64(24693)
	chainClient := kernel.ChainClient()
	probationList1, err := models.GetProbationListByEpoch(epochNum)
	require.NoError(err)
	probationList2, err := fetchProbationList(chainClient, epochNum)
	require.NoError(err)
	require.EqualValues(probationList1.ProbationList, probationList2.ProbationList)
	require.Equal(probationList1.IntensityRate, probationList2.IntensityRate)
}

func initTestConfig() (*gorm.DB, error) {
	_, err := config.New(os.Getenv("ConfigPath"))
	if err != nil {
		return nil, err
	}
	return db.Connect()
}
