package main

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestHermes(t *testing.T) {
	require := require.New(t)
	config.Default.Database = config.Database{
		Driver:   "postgres",
		Name:     "mainnet",
		Host:     "35.225.125.51",
		Port:     "5432",
		User:     "postgres",
		Password: "",
		Debug:    true,
	}

	_, err := db.Connect()
	require.NoError(err)
	for i := uint64(38256); i <= 25049; i++ {
		err = rebuildAccountRewardTable2(i)
	}
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
				return ErrConvertBigIntString
			}
			totalEpochReward, ok := big.NewInt(0).SetString(row.EpochReward, 10)
			if !ok {
				return ErrConvertBigIntString
			}
			totalFoundationBonus, ok := big.NewInt(0).SetString(row.FoundationBonus, 10)
			if !ok {
				return ErrConvertBigIntString
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
	_, err := db.LoadDBFromEnv()
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

func TestHermesUpdate(t *testing.T) {
	var candidateList *iotextypes.CandidateListV2
	var voteBucketList *iotextypes.VoteBucketList
	var probationList *iotextypes.ProbationCandidateList
	var err error
	require := require.New(t)
	_, err = db.LoadDBFromEnv()
	require.NoError(err)
	epochNumber := uint64(39282)
	blkHeight := kernel.GetEpochHeight(epochNumber)
	require.NoError(err)
	probationList, err = models.GetProbationListByEpoch(epochNumber)
	require.NoError(err)
	chainClient := kernel.ChainClient()
	voteBucketList, err = GetAllStakingBuckets(chainClient, kernel.GetEpochHeight(epochNumber-1))
	require.NoError(err)
	candidateList, err = models.GetCandidateList(epochNumber)
	require.NoError(err)
	if probationList != nil {
		candidateList, err = filterStakingCandidates(candidateList, probationList, blkHeight)
		require.NoError(err)
	}

	nameMap, err := ownerAddressToNameMap(candidateList)
	if err != nil {
		require.NoError(err)
	}
	pb := convertProbationListToLocal(probationList)
	intensityRate, probationMap := stakingProbationListToMap(candidateList, pb)
	//update aggregate voting table
	sumOfWeightedVotes := make(map[aggregateKey]*big.Int)
	totalVoted := big.NewInt(0)
	selfStakeIndex := selfStakeIndexMap(candidateList)
	lsdBuckets := make([]*iotextypes.VoteBucket, 0)
	for _, vote := range voteBucketList.Buckets {
		if vote.Index != 83 {
			continue
		}
		fmt.Printf("%+v\n", vote)
		if _, ok := nameMap[vote.CandidateAddress]; !ok {
			// the candidate is no longer active (and non-eligible for reward)
			// vote is not counted
			continue
		}
		//lsd buckets
		// if vote.ContractAddress != "" {
		// 	lsdBuckets = append(lsdBuckets, vote)
		// 	continue
		// }

		//for sumOfWeightedVotes
		key := aggregateKey{
			epochNumber:   epochNumber,
			candidateName: vote.CandidateAddress,
			voterAddress:  vote.Owner,
			isNative:      true,
		}
		selfStake := false
		if _, ok := selfStakeIndex[vote.Index]; ok {
			selfStake = true
		}
		weightedAmount, err := CalculateVoteWeight(GenesisVoteWeightCalConsts, vote, selfStake)
		require.NoError(err)
		stakeAmount, ok := big.NewInt(0).SetString(vote.StakedAmount, 10)
		require.True(ok)
		if val, ok := sumOfWeightedVotes[key]; ok {
			val.Add(val, weightedAmount)
		} else {
			sumOfWeightedVotes[key] = weightedAmount
		}
		totalVoted.Add(totalVoted, stakeAmount)
	}
	for _, vote := range lsdBuckets {
		key := aggregateKey{
			epochNumber:   epochNumber,
			candidateName: vote.CandidateAddress,
			voterAddress:  vote.Owner,
			isNative:      false,
		}
		selfStake := false
		fmt.Printf("lsdBuckets aggregateKey%+v\n", key)
		stakeAmount, ok := big.NewInt(0).SetString(vote.StakedAmount, 10)
		require.True(ok)
		weightedAmount := stakeAmount
		if config.Default.Genesis.RedseaBlockHeight <= blkHeight {
			weightedAmount, err = CalculateVoteWeight(GenesisVoteWeightCalConsts, vote, selfStake)
			require.NoError(err)
		}
		if val, ok := sumOfWeightedVotes[key]; ok {
			val.Add(val, weightedAmount)
		} else {
			sumOfWeightedVotes[key] = weightedAmount
		}
		totalVoted.Add(totalVoted, stakeAmount)
	}
	//update voting meta table
	totalWeighted := big.NewInt(0)
	for _, cand := range candidateList.Candidates {
		totalWeightedVotes, ok := big.NewInt(0).SetString(cand.TotalWeightedVotes, 10)
		if !ok {
			return
		}
		totalWeighted.Add(totalWeighted, totalWeightedVotes)
	}
	m := models.HermesVotingMeta{
		EpochNumber:        epochNumber,
		VotedToken:         decimal.NewFromBigInt(totalVoted, 0),
		TotalWeightedVotes: decimal.NewFromBigInt(totalWeighted, 0),
		DelegateCount:      len(candidateList.Candidates),
	}
	fmt.Printf("HermesVotingMeta %+v\n", m)

	uniqueMap := make(map[string]bool)
	batches := make([]models.HermesAggregateVoting, 0)
	for key, val := range sumOfWeightedVotes {
		k := fmt.Sprintf("%d%s%s%t", key.epochNumber, key.candidateName, key.voterAddress, key.isNative)

		if _, ok := uniqueMap[k]; ok {
			continue
		}
		if _, ok := probationMap[key.candidateName]; ok {
			// filter based on probation
			votingPower := new(big.Float).SetInt(val)
			val, _ = votingPower.Mul(votingPower, big.NewFloat(intensityRate)).Int(nil)
		}
		_, ok := nameMap[key.candidateName]
		require.True(ok)

		aggregateVotes := decimal.NewFromBigInt(val, 0)

		m := models.HermesAggregateVoting{
			EpochNumber:    key.epochNumber,
			CandidateName:  nameMap[key.candidateName],
			VoterAddress:   key.voterAddress,
			NativeFlag:     key.isNative,
			AggregateVotes: aggregateVotes,
		}
		fmt.Printf("HermesAggregateVoting %+v\n", m)
		batches = append(batches, m)
		uniqueMap[k] = true
	}
}

func TestHermesVotingResults(t *testing.T) {
	var candidateList *iotextypes.CandidateListV2
	// var voteBucketList *iotextypes.VoteBucketList
	var probationList *iotextypes.ProbationCandidateList
	var err error
	require := require.New(t)
	_, err = db.LoadDBFromEnv()
	require.NoError(err)
	epochNumber := uint64(39282)
	blkHeight := kernel.GetEpochHeight(epochNumber)
	epochStartheight := blkHeight
	chainClient, err := kernel.ChainClientWithEndPoint("api.iotex.one:80", true)
	require.NoError(err)
	probationList, err = models.GetProbationListByEpoch(epochNumber)
	require.NoError(err)
	// voteBucketList, err = models.GetVoteBucketList(epochNumber)
	// require.NoError(err)
	candidateList, err = models.GetCandidateList(epochNumber)
	require.NoError(err)
	if probationList != nil {
		candidateList, err = filterStakingCandidates(candidateList, probationList, blkHeight)
		require.NoError(err)
	}
	blockRewardPortionMap, epochRewardPortionMap, foundationBonusPortionMap, err := getAllStakingDelegateRewardPortions(epochStartheight, epochNumber, chainClient)
	require.NoError(err)

	for _, candidate := range candidateList.Candidates {

		blockRewardPortion := blockRewardPortionMap[candidate.OwnerAddress]
		epochRewardPortion := epochRewardPortionMap[candidate.OwnerAddress]
		foundationBonusPortion := foundationBonusPortionMap[candidate.OwnerAddress]
		encodedName := candidate.Name
		require.NoError(err)

		totalWeightedVotes, _ := decimal.NewFromString(candidate.TotalWeightedVotes)
		selfStakingTokens, _ := decimal.NewFromString(candidate.SelfStakingTokens)

		m := models.HermesVotingResult{
			EpochNumber:               epochNumber,
			DelegateName:              encodedName,
			OperatorAddress:           candidate.OperatorAddress,
			RewardAddress:             candidate.RewardAddress,
			StakingAddress:            candidate.OwnerAddress,
			TotalWeightedVotes:        totalWeightedVotes,
			SelfStaking:               selfStakingTokens,
			BlockRewardPercentage:     decimal.NewFromFloat(blockRewardPortion),
			EpochRewardPercentage:     decimal.NewFromFloat(epochRewardPortion),
			FoundationBonusPercentage: decimal.NewFromFloat(foundationBonusPortion),
		}
		fmt.Printf("%+v\n", m)
	}
}

func TestVotingResultV1(t *testing.T) {
	require := require.New(t)
	_, err := db.LoadDBFromEnv()
	require.NoError(err)
	epochNumber := uint64(25049)
	blkHeight := kernel.GetEpochHeight(epochNumber)
	prevEpochHeight := kernel.GetEpochHeight(epochNumber - 1)
	epochStartheight := blkHeight
	chainClient, err := kernel.ChainClientWithEndPoint("api.iotex.one:80", true)
	require.NoError(err)
	probationList, err := fetchProbationList(chainClient, epochNumber)
	require.NoError(err)
	candidateList, err := GetAllStakingCandidates(chainClient, prevEpochHeight)
	require.NoError(err)
	if probationList != nil {
		candidateList, err = filterStakingCandidates(candidateList, probationList, epochStartheight)
		require.NoError(err)
	}
	blockRewardPortionMap, epochRewardPortionMap, foundationBonusPortionMap, err := getAllStakingDelegateRewardPortions(epochStartheight, epochNumber, chainClient)
	require.NoError(err)
	fmt.Printf("blockRewardPortionMap = %v\n", blockRewardPortionMap)
	for _, candidate := range candidateList.Candidates {

		blockRewardPortion := blockRewardPortionMap[candidate.OwnerAddress]
		epochRewardPortion := epochRewardPortionMap[candidate.OwnerAddress]
		foundationBonusPortion := foundationBonusPortionMap[candidate.OwnerAddress]
		encodedName := candidate.Name
		require.NoError(err)

		totalWeightedVotes, _ := decimal.NewFromString(candidate.TotalWeightedVotes)
		selfStakingTokens, _ := decimal.NewFromString(candidate.SelfStakingTokens)

		m := models.HermesVotingResult{
			EpochNumber:               epochNumber,
			DelegateName:              encodedName,
			OperatorAddress:           candidate.OperatorAddress,
			RewardAddress:             candidate.RewardAddress,
			StakingAddress:            candidate.OwnerAddress,
			TotalWeightedVotes:        totalWeightedVotes,
			SelfStaking:               selfStakingTokens,
			BlockRewardPercentage:     decimal.NewFromFloat(blockRewardPortion),
			EpochRewardPercentage:     decimal.NewFromFloat(epochRewardPortion),
			FoundationBonusPercentage: decimal.NewFromFloat(foundationBonusPortion),
		}
		fmt.Printf("%+v\n", m)
	}
}
