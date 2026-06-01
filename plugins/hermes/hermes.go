package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.4.7"

const hermesBatchSize = 256

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

type hermesEpochContext struct {
	blkHeight      uint64
	epochNum       uint64
	chainClient    iotexapi.APIServiceClient
	candidateList  *iotextypes.CandidateListV2
	voteBucketList *iotextypes.VoteBucketList
	probationList  *iotextypes.ProbationCandidateList
}

func (b hermesPlugin) Name() string {
	return "hermes"
}

func (b hermesPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b hermesPlugin) DependentPlugins() []string {
	return []string{"block_meta", "block_reward", "probation", "candidatelist", "block_receipts"}
}

func (b hermesPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(),
		&models.HermesDistribute{},
		&models.HermesAggregateVoting{},
		&models.HermesVotingResult{},
		&models.HermesAccountReward{},
		&models.HermesVotingMeta{},
		&models.HermesBucketVoting{},
	); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b hermesPlugin) BatchSize() int {
	return hermesBatchSize
}

func (b hermesPlugin) buildDistributeBatch(blk *block.Block) ([]models.HermesDistribute, error) {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)
	batches := make([]models.HermesDistribute, 0)
	for _, receipt := range blk.Receipts {
		if receipt.Status != successStatus {
			continue
		}
		for _, log := range receipt.Logs() {
			topics := log.Topics
			if log.Address == "" || len(topics) < 2 || log.Address != HermesContractAddress {
				continue
			}
			if topics[0] != DISTRIBUTE {
				continue
			}
			event := struct {
				StartEpoch      *big.Int
				EndEpoch        *big.Int
				DelegateName    [32]byte
				NumOfRecipients *big.Int
				TotalAmount     *big.Int
			}{}
			if err := hermesABI.UnpackIntoInterface(&event, "Distribute", log.Data); err != nil {
				return nil, err
			}
			delegateNameTopic := log.Topics[1]
			delegateName := getDelegateNameFromTopic(delegateNameTopic)
			batches = append(batches, models.HermesDistribute{
				BlockHeight:     blkHeight,
				EpochNumber:     epochNum,
				ActionHash:      hex.EncodeToString(receipt.ActionHash[:]),
				StartEpoch:      event.StartEpoch.Uint64(),
				EndEpoch:        event.EndEpoch.Uint64(),
				DelegateName:    delegateName,
				NumOfRecipients: event.NumOfRecipients.Uint64(),
				TotalAmount:     decimal.NewFromBigInt(event.TotalAmount, 0),
			})
		}
	}
	return batches, nil
}

func (b hermesPlugin) prepareEpochContext(blkHeight uint64) (*hermesEpochContext, error) {
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	fairbankEpochNum := kernel.GetEpochNum(kernel.FairbankEffectiveHeight())
	if epochNum <= fairbankEpochNum || blkHeight != epochHeight || blkHeight < kernel.FairbankEffectiveHeight() {
		return nil, nil
	}

	if err := db.DB().Transaction(func(tx *gorm.DB) error {
		return rebuildAccountRewardTable(tx, epochNum-1)
	}); err != nil {
		return nil, errors.Wrap(err, "failed to rebuild account reward table")
	}

	chainClient := kernel.ChainClient()
	probationList, err := models.GetProbationListByEpoch(epochNum)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get probation list from chain service in epoch %d", epochNum)
	}
	preEpochNum := epochNum - 1
	voteBucketList, err := GetAllStakingBuckets(chainClient, kernel.GetEpochHeight(preEpochNum))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get buckets count")
	}
	candidateList, err := models.GetCandidateList(preEpochNum)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get candidates count")
	}
	if probationList != nil {
		candidateList, err = filterStakingCandidates(candidateList, probationList, blkHeight)
		if err != nil {
			return nil, errors.Wrap(err, "failed to filter candidate with probation list")
		}
	}
	return &hermesEpochContext{
		blkHeight:      blkHeight,
		epochNum:       epochNum,
		chainClient:    chainClient,
		candidateList:  candidateList,
		voteBucketList: voteBucketList,
		probationList:  probationList,
	}, nil
}

func (b hermesPlugin) commitDistributeBatch(tipHeight uint64, batches []models.HermesDistribute) error {
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(batches) > 0 {
			if err := tx.CreateInBatches(batches, 100).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b hermesPlugin) commitEpochBlock(epochCtx *hermesEpochContext, batches []models.HermesDistribute) error {
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(batches) > 0 {
			if err := tx.CreateInBatches(batches, 100).Error; err != nil {
				return err
			}
		}

		var count int64
		if err := tx.Model(&models.HermesVotingResult{}).Where("epoch_number = ?", epochCtx.epochNum).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := b.updateStakingResult(tx, epochCtx.candidateList, epochCtx.epochNum, epochCtx.blkHeight, epochCtx.chainClient); err != nil {
				return err
			}
		}

		if err := tx.Model(&models.HermesAggregateVoting{}).Where("epoch_number = ?", epochCtx.epochNum).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := b.updateAggregateStaking(epochCtx.blkHeight, tx, epochCtx.voteBucketList, epochCtx.candidateList, epochCtx.epochNum, epochCtx.probationList); err != nil {
				return err
			}
		}

		if err := tx.Model(&models.HermesBucketVoting{}).Where("epoch_number = ?", epochCtx.epochNum).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := b.updateBucketStaking(epochCtx.blkHeight, tx, epochCtx.voteBucketList, epochCtx.candidateList, epochCtx.epochNum, epochCtx.probationList); err != nil {
				return err
			}
		}

		return db.UpdateIndexHeightByTx(tx, b.Name(), epochCtx.blkHeight)
	})
}

func (b hermesPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return b.PutBlocks(ctx, []*block.Block{blk})
}

func (b hermesPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	if len(blks) == 0 {
		return nil
	}

	pending := make([]models.HermesDistribute, 0)
	pendingHeight := uint64(0)
	flush := func() error {
		if pendingHeight == 0 {
			return nil
		}
		err := b.commitDistributeBatch(pendingHeight, pending)
		pending = nil
		pendingHeight = 0
		return err
	}

	for _, blk := range blks {
		batches, err := b.buildDistributeBatch(blk)
		if err != nil {
			return err
		}
		epochCtx, err := b.prepareEpochContext(blk.Height())
		if err != nil {
			return err
		}
		if epochCtx != nil {
			if err := flush(); err != nil {
				return err
			}
			if err := b.commitEpochBlock(epochCtx, batches); err != nil {
				return err
			}
			continue
		}
		pending = append(pending, batches...)
		pendingHeight = blk.Height()
	}

	return flush()
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
			StakingAddress:            candidate.Id,
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
func (b hermesPlugin) updateAggregateStaking(blkHeight uint64, tx *gorm.DB, votes *iotextypes.VoteBucketList, delegates *iotextypes.CandidateListV2, epochNumber uint64, probationList *iotextypes.ProbationCandidateList) (err error) {
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
	lsdBuckets := make([]*iotextypes.VoteBucket, 0)
	for _, vote := range votes.Buckets {
		if _, ok := nameMap[vote.CandidateAddress]; !ok {
			// the candidate is no longer active (and non-eligible for reward)
			// vote is not counted
			continue
		}
		//lsd buckets
		if vote.ContractAddress != "" {
			lsdBuckets = append(lsdBuckets, vote)
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
	for _, vote := range lsdBuckets {
		key := aggregateKey{
			epochNumber:   epochNumber,
			candidateName: vote.CandidateAddress,
			voterAddress:  vote.Owner,
			isNative:      false,
		}
		selfStake := false

		stakeAmount, ok := big.NewInt(0).SetString(vote.StakedAmount, 10)
		if !ok {
			return errors.New("failed to convert string to big int")
		}
		weightedAmount := stakeAmount
		if blkHeight >= config.Default.Genesis.RedseaBlockHeight {
			weightedAmount, err = CalculateVoteWeight(GenesisVoteWeightCalConsts, vote, selfStake)
			if err != nil {
				return errors.Wrap(err, "failed to calculate vote weight")
			}
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
	m := models.HermesVotingMeta{
		EpochNumber:        epochNumber,
		VotedToken:         decimal.NewFromBigInt(totalVoted, 0),
		TotalWeightedVotes: decimal.NewFromBigInt(totalWeighted, 0),
		DelegateCount:      len(delegates.Candidates),
	}
	if err := tx.Create(&m).Error; err != nil {
		return errors.Wrap(err, "failed to create voting meta")
	}

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
		batches = append(batches, m)
		uniqueMap[k] = true
	}
	return tx.CreateInBatches(batches, 100).Error
}

func (b hermesPlugin) updateBucketStaking(blkHeight uint64, tx *gorm.DB, votes *iotextypes.VoteBucketList, delegates *iotextypes.CandidateListV2, epochNumber uint64, probationList *iotextypes.ProbationCandidateList) (err error) {
	nameMap, err := ownerAddressToNameMap(delegates)
	if err != nil {
		return errors.Wrap(err, "owner address to name map error")
	}
	pb := convertProbationListToLocal(probationList)
	intensityRate, probationMap := stakingProbationListToMap(delegates, pb)
	selfStakeIndex := selfStakeIndexMap(delegates)
	lsdBuckets := make([]*iotextypes.VoteBucket, 0)
	bucketBatches := make([]models.HermesBucketVoting, 0)
	for _, vote := range votes.Buckets {
		if _, ok := nameMap[vote.CandidateAddress]; !ok {
			// the candidate is no longer active (and non-eligible for reward)
			// vote is not counted
			continue
		}
		//lsd buckets
		if vote.ContractAddress != "" {
			lsdBuckets = append(lsdBuckets, vote)
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

		if _, ok := probationMap[key.candidateName]; ok {
			// filter based on probation
			votingPower := new(big.Float).SetInt(weightedAmount)
			weightedAmount, _ = votingPower.Mul(votingPower, big.NewFloat(intensityRate)).Int(nil)
		}
		bucketBatches = append(bucketBatches, models.HermesBucketVoting{
			EpochNumber:   key.epochNumber,
			CandidateName: nameMap[key.candidateName],
			VoterAddress:  key.voterAddress,
			NativeFlag:    key.isNative,
			BucketID:      vote.Index,
			Votes:         decimal.NewFromBigInt(weightedAmount, 0),
		})
	}
	for _, vote := range lsdBuckets {
		key := aggregateKey{
			epochNumber:   epochNumber,
			candidateName: vote.CandidateAddress,
			voterAddress:  vote.Owner,
			isNative:      false,
		}
		selfStake := false

		stakeAmount, ok := big.NewInt(0).SetString(vote.StakedAmount, 10)
		if !ok {
			return errors.New("failed to convert string to big int")
		}
		weightedAmount := stakeAmount
		if blkHeight >= config.Default.Genesis.RedseaBlockHeight {
			weightedAmount, err = CalculateVoteWeight(GenesisVoteWeightCalConsts, vote, selfStake)
			if err != nil {
				return errors.Wrap(err, "failed to calculate vote weight")
			}
		}

		if _, ok := probationMap[key.candidateName]; ok {
			// filter based on probation
			votingPower := new(big.Float).SetInt(weightedAmount)
			weightedAmount, _ = votingPower.Mul(votingPower, big.NewFloat(intensityRate)).Int(nil)
		}
		bucketBatches = append(bucketBatches, models.HermesBucketVoting{
			EpochNumber:   key.epochNumber,
			CandidateName: nameMap[key.candidateName],
			VoterAddress:  key.voterAddress,
			NativeFlag:    key.isNative,
			BucketID:      vote.Index,
			Votes:         decimal.NewFromBigInt(weightedAmount, 0),
		})
	}

	return tx.CreateInBatches(bucketBatches, 100).Error
}

func (b hermesPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b hermesPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = hermesPlugin{}
