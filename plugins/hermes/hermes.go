package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.3.1"

var FairbankBlockHeight = 5165641

const (
	HermesContractAddress = "io1fqulsuv8p820wmr0yd39jzx0m3pnpmuzzcywh8"
)

var DISTRIBUTE hash.Hash256
var successStatus = uint64(1)

var (
	hermesABI          abi.ABI
	delegateProfileABI abi.ABI
)

func initAddress() error {
	var err error
	//Distribute(uint256,uint256,bytes32,uint256,uint256)
	DISTRIBUTE, err = hash.HexStringToHash256("7de680eab607fdcc6137464e40d375ad63446cf255dcea9bd4a19676f7f24f56")
	if err != nil {
		return err
	}
	hermesABI, err = abi.JSON(strings.NewReader(HermesABI))
	if err != nil {
		return err
	}
	delegateProfileABI, err = abi.JSON((strings.NewReader(DelegateProfileABI)))
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

func (b hermesPlugin) DependentPlugin() string {
	return "block_meta"
}

func (b hermesPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.DB().AutoMigrate(
		&models.HermesDistribute{},
		&models.HermesAggregateVoting{},
		&models.HermesVotingResult{},
		&models.HermesAccountReward{},
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
	for _, receipt := range blk.Receipts {
		if receipt.Status != successStatus {
			continue
		}
		for _, log := range receipt.Logs() {
			topics := log.Topics
			if log.Address == "" || len(topics) < 2 {
				continue
			}
			switch log.Address {
			case HermesContractAddress:
				/**
				 * Distribute(uint256 startEpoch, uint256 endEpoch, bytes32 indexed delegateName, uint256 numOfRecipients, uint256 totalAmount);
				 */
				switch topics[0] {
				case DISTRIBUTE:
					event := struct {
						StartEpoch      *big.Int
						EndEpoch        *big.Int
						DelegateName    [32]byte
						NumOfRecipients *big.Int
						TotalAmount     *big.Int
					}{}
					err := hermesABI.UnpackIntoInterface(&event, "Distribute", log.Data)
					if err != nil {
						return err
					}
					delegateNameTopic := log.Topics[1]
					delegateName := getDelegateNameFromTopic(delegateNameTopic)
					m := models.HermesDistribute{
						BlockHeight:     blkHeight,
						StartEpoch:      event.StartEpoch.Uint64(),
						EndEpoch:        event.EndEpoch.Uint64(),
						DelegateName:    delegateName,
						NumOfRecipients: event.NumOfRecipients.Uint64(),
						TotalAmount:     decimal.NewFromBigInt(event.TotalAmount, 0),
					}
					if err = db.DB().Create(&m).Error; err != nil {
						return err
					}
				}
			}
		}
	}
	if blkHeight == epochHeight && blkHeight >= kernel.FairbankEffectiveHeight() {
		var count int64
		if err := rebuildAccountRewardTable(epochNum - 1); err != nil {
			return errors.Wrap(err, "failed to rebuild account reward table")
		}

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
			return errors.Wrap(err, "failed to get candidates count")
		}
		if probationList != nil {
			candidateList, err = filterStakingCandidates(candidateList, probationList, blkHeight)
			if err != nil {
				return errors.Wrap(err, "failed to filter candidate with probation list")
			}
		}
		err = db.DB().Transaction(func(tx *gorm.DB) error {
			// update voting_result table
			err := tx.Model(&models.HermesVotingResult{}).Where("epoch_number = ?", epochNum).Count(&count).Error
			if err != nil {
				return err
			}
			if count == 0 {
				if err = b.updateStakingResult(tx, candidateList, epochNum, blkHeight, chainClient); err != nil {
					return err
				}
			}
			err = tx.Model(&models.HermesAggregateVoting{}).Where("epoch_number = ?", epochNum).Count(&count).Error
			if err != nil {
				return err
			}
			if count == 0 {
				// update aggregate_voting and voting_meta table
				if err = b.updateAggregateStaking(tx, voteBucketList, candidateList, epochNum, probationList); err != nil {
					return err
				}
			}
			return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
		})
		if err != nil {
			return err
		}
	}
	return db.UpdateIndexHeight(b.Name(), blk.Height())
}

func (b hermesPlugin) updateStakingResult(tx *gorm.DB, candidates *iotextypes.CandidateListV2, epochNumber, epochStartheight uint64, chainClient iotexapi.APIServiceClient) (err error) {

	blockRewardPortionMap, epochRewardPortionMap, foundationBonusPortionMap, err := getAllStakingDelegateRewardPortions(epochStartheight, epochNumber, chainClient)
	if err != nil {
		return errors.Errorf("get delegate reward portions:%d,%s", epochStartheight, err.Error())
	}

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

	uniqueMap := make(map[string]bool)
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

		m := models.HermesAggregateVoting{
			EpochNumber:    key.epochNumber,
			CandidateName:  nameMap[key.candidateName],
			VoterAddress:   key.voterAddress,
			NativeFlag:     key.isNative,
			AggregateVotes: aggregateVotes,
		}
		if err = tx.Create(&m).Error; err != nil {
			return err
		}
		uniqueMap[k] = true
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
