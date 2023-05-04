package main

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestGetDelegateActive(t *testing.T) {
	require := require.New(t)
	_, err := db.LoadDBFromEnv()
	require.NoError(err)
	getDelegateActive(15892200)
}

func TestVotes(t *testing.T) {
	require := require.New(t)
	_, err := db.LoadDBFromEnv()
	require.NoError(err)

	/*
		10250  gamefantasy error
		epochData:{num:10249 height:5562361}
		epochData:{num:10250 height:5563081}
	*/
	fmt.Printf("%d=", kernel.GetEpochLastBlockHeight(10250))
	return
	epochNumber := uint64(10249)
	pluginHeight := kernel.GetEpochLastBlockHeight(epochNumber)
	stakings, err := getCandidateStaking(pluginHeight)
	require.NoError(err)
	delegateActives := getDelegateActive(pluginHeight)
	candidates, err := models.GetAllCandidates()
	require.NoError(err)
	delegateMap := make(map[string]*Delegate)
	totalVotes := big.NewInt(0)
	tmpStakings := make([]Staking, 0)
	for _, staking := range stakings {
		cand, err := candidates.ByOwnerAddress(staking.Candidate)
		require.NoError(err)
		if cand.Name != "gamefantasy" {
			continue
		}
		tmpStakings = append(tmpStakings, *staking)
		delegate, ok := delegateMap[staking.Candidate]
		if !ok {

			active := false
			productionNum := 0
			//cand.OperatorAddress is block producer address
			if productivity, ok := delegateActives[cand.OperatorAddress]; ok {
				active = true
				productionNum = productivity
			}
			delegate = &Delegate{
				Name:            cand.Name,
				OwnerAddress:    staking.OwnerAddress,
				Candidate:       staking.Candidate,
				OperatorAddress: cand.OperatorAddress,
				RewardAddress:   cand.RewardAddress,
				Active:          active,
				StakeAmount:     big.NewInt(0),
				VoteWeight:      big.NewInt(0),
				SelfStake:       isSelfStake(staking.Candidate, epochNumber),
				Productivity:    productionNum,
			}
		}
		stakeAmount, _ := big.NewInt(0).SetString(staking.Amount, 0)
		delegate.StakeAmount = delegate.StakeAmount.Add(delegate.StakeAmount, stakeAmount)
		voteBucket := &VoteBucket{
			StakedAmount:   stakeAmount,
			AutoStake:      staking.AutoStake,
			StakedDuration: staking.Duration,
		}
		selfAutoStake := false
		if staking.OwnerAddress == staking.Candidate {
			selfAutoStake = true
		}
		voteWeight := calculateVoteWeight(config.Default.Genesis.Staking.VoteWeightCalConsts, voteBucket, selfAutoStake)
		delegate.VoteWeight = delegate.VoteWeight.Add(delegate.VoteWeight, voteWeight)
		totalVotes = totalVotes.Add(totalVotes, voteWeight)
		delegateMap[staking.Candidate] = delegate
	}
	fmt.Printf("%+v\n", tmpStakings)
	probationList := getProbationList(pluginHeight)
	for c, d := range delegateMap {
		probated := false
		if _, ok := probationList[d.OperatorAddress]; ok {
			probated = true
		}
		if d.Name != "gamefantasy" {
			continue
		}

		modelDelegate := models.Delegate{
			BlockHeight:     pluginHeight,
			OperatorAddress: d.OperatorAddress,
			RewardAddress:   d.RewardAddress,
			OwnerAddress:    d.OwnerAddress,
			Candidate:       c,
			Active:          d.Active,
			Name:            d.Name,
			StakeAmount:     decimal.NewFromBigInt(d.StakeAmount, 0),
			VoteWeight:      decimal.NewFromBigInt(d.VoteWeight, 0),
			SelfStake:       d.SelfStake,
			Productivity:    d.Productivity,
			Probated:        probated,
		}
		fmt.Printf("%+v\n", modelDelegate)
	}
}
