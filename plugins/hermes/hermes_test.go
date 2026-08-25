package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

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

func TestGetLogs(t *testing.T) {
	t.Skip("skip test")
	r := require.New(t)
	// init db
	config.Default.Database.Driver = "postgres"
	config.Default.Database.User = ""
	config.Default.Database.Password = ""
	config.Default.Database.Host = ""
	config.Default.Database.Port = "5432"
	config.Default.Database.Name = "mainnet"
	_, err := db.Connect()
	r.NoError(err)
	// init grpc
	var opt grpc.DialOption
	endpoint := "api.iotex.one:443"
	opt = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	conn, err := grpc.Dial(endpoint, grpc.WithTimeout(60*time.Second), opt, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(32*10e6)))
	if err != nil {
		log.L().Error("failed to connect to chain endpoint.",
			zap.Error(err),
			zap.String("endpoint", endpoint),
		)
	}
	chainClient := iotexapi.NewAPIServiceClient(conn)
	resp, err := chainClient.GetChainMeta(context.Background(), &iotexapi.GetChainMetaRequest{})
	r.NoError(err)
	end := resp.ChainMeta.Height

	tp, err := hex.DecodeString(topicProfileUpdated)
	r.NoError(err)
	topics := [][]byte{tp}
	contractAddress := RewardPortionContract
	check := func(from, count uint64) int {
		logsFromDB, err := getLogFromDB(contractAddress, topics, uint64(from), count)
		r.NoError(err)
		response, err := chainClient.GetLogs(context.Background(), &iotexapi.GetLogsRequest{
			Filter: &iotexapi.LogsFilter{
				Address: []string{contractAddress},
				Topics:  []*iotexapi.Topics{{Topic: topics}},
			},
			Lookup: &iotexapi.GetLogsRequest_ByRange{
				ByRange: &iotexapi.GetLogsByRange{
					FromBlock: from,
					ToBlock:   from + count,
				},
			},
		})
		r.NoError(err)
		logsFromChain := response.GetLogs()

		r.Len(logsFromDB, len(logsFromChain), "logs from db and chain should have the same length")
		// compare := func(li, lj *iotextypes.Log) bool {
		// 	if li.BlkHeight != lj.BlkHeight {
		// 		return li.BlkHeight < lj.BlkHeight
		// 	}
		// 	if li.TxIndex != lj.TxIndex {
		// 		return li.TxIndex < lj.TxIndex
		// 	}
		// 	if !bytes.Equal(li.ActHash, lj.ActHash) {
		// 		return bytes.Compare(li.ActHash, lj.ActHash) < 0
		// 	}
		// 	if li.Index != lj.Index {
		// 		return li.Index < lj.Index
		// 	}
		// 	return bytes.Compare(li.Data, lj.Data) < 0
		// }
		// sort.Slice(logsFromChain, func(i, j int) bool {
		// 	return compare(logsFromChain[i], logsFromChain[j])
		// })
		// sort.Slice(logsFromDB, func(i, j int) bool {
		// 	return compare(logsFromDB[i], logsFromDB[j])
		// })
		for i, logFromDB := range logsFromDB {
			logFromChain := logsFromChain[i]

			r.Equal(logFromDB.BlkHeight, logFromChain.BlkHeight, "block height should be the same")
			// r.Equal(logFromDB.Index, logFromChain.Index, "index should be the same")
			// r.Equal(logFromDB.TxIndex, logFromChain.TxIndex, "tx index should be the same")
			r.Equal(logFromDB.ActHash, logFromChain.ActHash, "action hash should be the same")
			r.Equal(logFromDB.ContractAddress, logFromChain.ContractAddress, "contract address should be the same")
			r.Equal(logFromDB.Topics, logFromChain.Topics, "topics should be the same")
			r.Equal(logFromDB.Data, logFromChain.Data, "data should be the same")
		}
		return len(logsFromDB)
	}

	from := uint64(RewardportionContractDeployHeight)
	count := uint64(maxBlockRange)
	for from <= end {
		size := check(from, count)
		t.Log("verified from", from, "to", from+count, "size", size)
		from += count
	}
}
