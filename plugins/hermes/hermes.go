package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	"github.com/iotexproject/iotex-core/ioctl/util"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.1.0"

var FairbankBlockHeight = 5165641

const (
	HermesContractAddress = "io1fqulsuv8p820wmr0yd39jzx0m3pnpmuzzcywh8"
)

var DISTRIBUTE hash.Hash256

func initAddress() error {
	var err error
	//Distribute(uint256,uint256,bytes32,uint256,uint256)
	DISTRIBUTE, err = hash.HexStringToHash256("7de680eab607fdcc6137464e40d375ad63446cf255dcea9bd4a19676f7f24f56")
	if err != nil {
		return err
	}
	return nil
}

type hermesPlugin struct {
}

func (b hermesPlugin) Name() string {
	return "hermes"
}

func (b hermesPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b hermesPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.DB().AutoMigrate(
		&models.HermesDistribute{},
		&models.HermesAggregateVoting{},
		&models.HermesVotingResult{},
		&models.HermesVotingResult2{},
	); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b hermesPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	chainClient := kernel.ChainClient()
	if blkHeight == epochHeight && blkHeight > genesis.Default.Blockchain.FairbankBlockHeight {
		probationList, err := fetchProbationList(chainClient, epochNum)
		if err != nil {
			return errors.Wrapf(err, "failed to get probation list from chain service in epoch %d", epochNum)
		}
		prevEpochHeight := kernel.GetEpochHeight(epochNum - 1)
		voteBucketList, err := GetAllStakingBuckets(chainClient, prevEpochHeight)
		if err != nil {
			return errors.Wrap(err, "failed to get buckets count")
		}
		candidateList, err := GetAllStakingCandidates(chainClient, prevEpochHeight)
		if err != nil {
			return errors.Wrap(err, "failed to get buckets count")
		}

		err = db.DB().Transaction(func(tx *gorm.DB) error {
			// update voting_result table
			if err = b.updateStakingResult(tx, candidateList, epochNum, blkHeight, chainClient); err != nil {
				return err
			}
			// update aggregate_voting and voting_meta table
			if err = b.updateAggregateStaking(tx, voteBucketList, candidateList, epochNum, probationList); err != nil {
				return err
			}
			return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
		})
		return err
	}

	return db.UpdateIndexHeight(b.Name(), blk.Height())
}

func (b hermesPlugin) updateStakingResult(tx *gorm.DB, candidates *iotextypes.CandidateListV2, epochNumber, epochStartheight uint64, chainClient iotexapi.APIServiceClient) (err error) {

	blockRewardPortionMap, epochRewardPortionMap, foundationBonusPortionMap, err := getAllStakingDelegateRewardPortions(epochStartheight, epochNumber, chainClient)
	if err != nil {
		return errors.Errorf("get delegate reward portions:%d,%s", epochStartheight, err.Error())
	}

	fmt.Printf("%v epochNumber=%d", candidates.Candidates, epochNumber)

	for _, candidate := range candidates.Candidates {

		blockRewardPortion := blockRewardPortionMap[candidate.OwnerAddress]
		epochRewardPortion := epochRewardPortionMap[candidate.OwnerAddress]
		foundationBonusPortion := foundationBonusPortionMap[candidate.OwnerAddress]
		encodedName := candidate.Name
		if err != nil {
			return errors.Wrap(err, "encode delegate name error")
		}

		totalWeightedVotes, _ := decimal.NewFromString(candidate.TotalWeightedVotes)
		selfStakingTokens, _ := decimal.NewFromString(candidate.SelfStakingTokens)

		m := models.HermesVotingResult2{
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
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}
func (b hermesPlugin) updateAggregateStaking(tx *gorm.DB, votes *iotextypes.VoteBucketList, delegates *iotextypes.CandidateListV2, epochNumber uint64, probationList *iotextypes.ProbationCandidateList) (err error) {
	nameMap, err := ownerAddressToNameMap(delegates)
	if err != nil {
		return errors.Wrap(err, "owner address to name map error")
	}
	pb := convertProbationListToLocal(probationList)
	intensityRate, probationMap := stakingProbationListToMap(delegates, pb)
	//update aggregate voting table
	sumOfWeightedVotes := make(map[aggregateKey]*big.Int)
	totalVoted := big.NewInt(0)
	selfStakeIndex := selfStakeIndexMap(delegates)
	for _, vote := range votes.Buckets {
		if _, ok := nameMap[vote.CandidateAddress]; !ok {
			// the candidate is no longer active (and non-eligible for reward)
			// vote is not counted
			continue
		}
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
		if err != nil {
			return errors.Wrap(err, "failed to calculate vote weight")
		}
		stakeAmount, ok := big.NewInt(0).SetString(vote.StakedAmount, 10)
		if !ok {
			return errors.New("failed to convert string to big int")
		}
		if val, ok := sumOfWeightedVotes[key]; ok {
			val.Add(val, weightedAmount)
		} else {
			sumOfWeightedVotes[key] = weightedAmount
		}
		totalVoted.Add(totalVoted, stakeAmount)
	}

	for key, val := range sumOfWeightedVotes {
		if _, ok := probationMap[key.candidateName]; ok {
			// filter based on probation
			votingPower := new(big.Float).SetInt(val)
			val, _ = votingPower.Mul(votingPower, big.NewFloat(intensityRate)).Int(nil)
		}
		if _, ok := nameMap[key.candidateName]; !ok {
			return errors.New("candidate cannot find name through owner address")
		}
		voterAddress, err := util.IoAddrToEvmAddr(key.voterAddress)
		if err != nil {
			return errors.Wrap(err, "failed to convert IoTeX address to ETH address")
		}
		aggregateVotes := decimal.NewFromBigInt(val, 0)

		m := models.HermesAggregateVoting{
			EpochNumber:    key.epochNumber,
			CandidateName:  nameMap[key.candidateName],
			VoterAddress:   voterAddress.String(),
			NativeFlag:     key.isNative,
			AggregateVotes: aggregateVotes,
		}
		if err = tx.Create(&m).Error; err != nil {
			return err
		}
	}

	return
}
func (b hermesPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b hermesPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = hermesPlugin{}
