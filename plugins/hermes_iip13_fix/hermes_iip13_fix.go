package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "1.0.0"

var FairbankBlockHeight = 5165641

type hermesIIP13FixPlugin struct {
}

func (b hermesIIP13FixPlugin) Name() string {
	return "hermes_iip13_fix"
}

func (b hermesIIP13FixPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b hermesIIP13FixPlugin) DependentPlugins() []string {
	return []string{"probation", "candidatelist"}
}

func (b hermesIIP13FixPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(),
		&HermesAggregateVoting{},
		&HermesVotingMeta{},
	); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	height, err := db.GetIndexHeight(b.Name())
	if err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	if height == 0 {
		return db.UpdateIndexHeight(b.Name(), config.Default.Genesis.Poll.SystemStakingContractHeight)
	}
	return nil
}

func (b hermesIIP13FixPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	chainClient := kernel.ChainClient()
	var candidateList *iotextypes.CandidateListV2
	var voteBucketList *iotextypes.VoteBucketList
	var probationList *iotextypes.ProbationCandidateList
	var err error
	if blkHeight == epochHeight && blkHeight >= kernel.FairbankEffectiveHeight() {
		probationList, err = models.GetProbationListByEpoch(epochNum)
		if err != nil {
			return errors.Wrapf(err, "failed to get probation list from chain service in epoch %d", epochNum)
		}
		preEpochNum := epochNum - 1
		voteBucketList, err = GetAllStakingBuckets(chainClient, kernel.GetEpochHeight(preEpochNum))
		if err != nil {
			return errors.Wrap(err, "failed to get buckets count")
		}
		candidateList, err = models.GetCandidateList(preEpochNum)
		if err != nil {
			return errors.Wrap(err, "failed to get candidates count")
		}
		if probationList != nil {
			candidateList, err = filterStakingCandidates(candidateList, probationList, blkHeight)
			if err != nil {
				return errors.Wrap(err, "failed to filter candidate with probation list")
			}
		}
	}
	err = db.DB().Transaction(func(tx *gorm.DB) error {
		if blkHeight == epochHeight && blkHeight >= kernel.FairbankEffectiveHeight() {
			var count int64
			err = tx.Model(&HermesAggregateVoting{}).Where("epoch_number = ?", epochNum).Count(&count).Error
			if err != nil {
				return err
			}
			if count == 0 {
				// update aggregate_voting and voting_meta table
				if err = b.updateAggregateStaking(blkHeight, tx, voteBucketList, candidateList, epochNum, probationList); err != nil {
					return err
				}
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err
}

func (b hermesIIP13FixPlugin) updateAggregateStaking(blkHeight uint64, tx *gorm.DB, votes *iotextypes.VoteBucketList, delegates *iotextypes.CandidateListV2, epochNumber uint64, probationList *iotextypes.ProbationCandidateList) (err error) {
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
	epochNum := kernel.GetEpochNum(blkHeight)
	preEpochNum := epochNum - 1
	lsdBuckets, err := getLSDStakingBuckets(kernel.GetEpochHeight(preEpochNum))
	if err != nil {
		return errors.Wrap(err, "failed to get LSD staking buckets")
	}
	for _, bucket := range lsdBuckets {
		if _, ok := nameMap[bucket.DelegateOwnerAddress]; !ok {
			log.S().Warnf("the candidate is no longer active (and non-eligible for reward), vote is not counted, delegateOwnerAddress: %s", bucket.DelegateOwnerAddress)
			continue
		}
		key := aggregateKey{
			epochNumber:   epochNumber,
			candidateName: bucket.DelegateOwnerAddress,
			voterAddress:  bucket.OwnerAddress,
			isNative:      false,
		}
		selfStake := false

		if bucket.StakedAmount == "0" {
			continue
		}
		stakeAmount, ok := big.NewInt(0).SetString(bucket.StakedAmount, 10)
		if !ok {
			return errors.New("failed to convert string to big int")
		}
		vote := &iotextypes.VoteBucket{
			AutoStake:      bucket.AutoStake,
			StakedDuration: bucket.Duration,
			StakedAmount:   bucket.StakedAmount,
		}
		weightedAmount, err := CalculateVoteWeight(GenesisVoteWeightCalConsts, vote, selfStake)
		if err != nil {
			return errors.Wrap(err, "failed to calculate vote weight")
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
	for _, cand := range delegates.Candidates {
		totalWeightedVotes, ok := big.NewInt(0).SetString(cand.TotalWeightedVotes, 10)
		if !ok {
			err = errors.New("total weighted votes convert error")
			return
		}
		totalWeighted.Add(totalWeighted, totalWeightedVotes)
	}
	m := HermesVotingMeta{
		EpochNumber:        epochNumber,
		VotedToken:         decimal.NewFromBigInt(totalVoted, 0),
		TotalWeightedVotes: decimal.NewFromBigInt(totalWeighted, 0),
		DelegateCount:      len(delegates.Candidates),
	}
	if err := tx.Create(&m).Error; err != nil {
		return errors.Wrap(err, "failed to create voting meta")
	}

	uniqueMap := make(map[string]bool)
	batches := make([]HermesAggregateVoting, 0)
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
		if _, ok := nameMap[key.candidateName]; !ok {
			return errors.New("candidate cannot find name through owner address")
		}

		aggregateVotes := decimal.NewFromBigInt(val, 0)

		m := HermesAggregateVoting{
			EpochNumber:    key.epochNumber,
			CandidateName:  nameMap[key.candidateName],
			VoterAddress:   key.voterAddress,
			NativeFlag:     key.isNative,
			AggregateVotes: aggregateVotes,
		}
		batches = append(batches, m)
		uniqueMap[k] = true
	}
	return tx.CreateInBatches(batches, 100).Error
}
func (b hermesIIP13FixPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b hermesIIP13FixPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = hermesIIP13FixPlugin{}
