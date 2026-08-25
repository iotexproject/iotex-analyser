// Package hermes_reward rebuilds the per-delegate hermes_account_rewards
// rollup for a closed epoch by aggregating block_rewards and splitting each
// reward address across the delegates that share it.
//
// It lives here rather than in plugins/hermes so that both the plugin (which
// rebuilds an epoch as the indexer crosses its boundary) and the
// `tools backfill-hermes-rewards` command (which rebuilds a range after the
// fact) run the exact same code.
package hermes_reward

import (
	"math/big"
	"sort"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrEmptyRecords        = errors.New("empty records")
	ErrConvertBigIntString = errors.New("failed to convert string to big int")
)

type AggregateReward struct {
	EpochNumber     uint64
	RewardAddress   string
	BlockReward     string
	EpochReward     string
	FoundationBonus string
	PriorityBonus   string
}

type (
	CandidateVote struct {
		CandidateName      string
		TotalWeightedVotes *big.Int
	}
	Productivity struct {
		Production         uint64
		ExpectedProduction uint64
	}
	ProductivityHistory struct {
		EpochNumber        uint64
		ProducerName       string
		Production         uint64
		ExpectedProduction uint64
	}
)

// Coverage describes whether the indexer captured the whole of an epoch.
type Coverage struct {
	// OK is true when the epoch is safe to aggregate over.
	OK bool
	// Reason explains why it is not, when OK is false.
	Reason string
	// Blocks is how many block_meta rows the epoch has.
	Blocks int64
	// Earliest is the lowest indexed block height in the epoch, 0 if none.
	Earliest uint64
	// EpochStart is the epoch's first block height on chain.
	EpochStart uint64
}

// CheckEpochCoverage reports whether the indexer captured the whole of `epoch`.
// That is a precondition for rebuilding hermes_account_rewards: the rebuild
// SUMs block_rewards across the epoch, so a window we only partially indexed
// would silently under-count and the shortfall would be invisible downstream.
//
// The signal is block_meta, which carries one row per block whether or not
// that block paid anything out. It used to be block_rewards, which only
// carries a row when a GrantReward actually succeeded -- so the old check
// conflated "we did not index the start of this epoch" with "the chain did not
// pay a reward at the start of this epoch". Those two came apart on
// 2026-08-18, when the rewarding fund's available balance ran dry: grants
// began failing, the first block_rewards row of an epoch drifted tens of
// blocks past the epoch start, and every epoch from 63860 to 64002 was skipped
// as "partial" even though block_meta held all 1440 blocks. The skip is not
// retried -- the plugin only rebuilds an epoch as the indexer crosses its
// boundary, and the indexer never revisits a block -- so those rows stayed
// missing and Hermes blocked on "bookkeeping info doesn't exist for Epoch
// 63860" until they were backfilled by hand.
//
// A sparse block_rewards is now a legitimate outcome that gets aggregated as
// it stands; only a short block_meta means we are looking at an epoch the
// indexer joined midway (catch-up mode), which is the case worth skipping.
func CheckEpochCoverage(tx *gorm.DB, epoch uint64) (*Coverage, error) {
	var probe struct {
		Blocks   int64
		Earliest uint64
	}
	if err := tx.Model(&models.BlockMeta{}).
		Select("COUNT(*) AS blocks, COALESCE(MIN(block_height), 0) AS earliest").
		Where("epoch_num = ?", epoch).
		Scan(&probe).Error; err != nil {
		return nil, errors.Wrap(err, "failed to probe block_meta coverage")
	}

	cov := &Coverage{
		Blocks:     probe.Blocks,
		Earliest:   probe.Earliest,
		EpochStart: kernel.GetEpochHeight(epoch),
	}
	switch {
	case probe.Blocks == 0:
		cov.Reason = "no block_meta rows for epoch"
	case probe.Earliest > cov.EpochStart:
		cov.Reason = "block_meta starts after the epoch start"
	default:
		cov.OK = true
	}
	return cov, nil
}

// RebuildAccountRewardTable replaces hermes_account_rewards for lastEpoch with
// a fresh aggregation of block_rewards. It deletes the epoch's existing rows
// before inserting, so re-running it is idempotent.
func RebuildAccountRewardTable(tx *gorm.DB, lastEpoch uint64) error {
	if lastEpoch == 0 {
		return nil
	}
	cov, err := CheckEpochCoverage(tx, lastEpoch)
	if err != nil {
		return err
	}
	if !cov.OK {
		log.L().Warn("hermes: skipping account_reward rebuild, epoch not fully indexed",
			zap.Uint64("epoch", lastEpoch),
			zap.String("reason", cov.Reason),
			zap.Uint64("epochStart", cov.EpochStart),
			zap.Uint64("earliestBlockMeta", cov.Earliest),
			zap.Int64("blocks", cov.Blocks))
		return nil
	}
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
	if err := tx.Raw("SELECT epoch_number, reward_address, SUM(block_reward) block_reward, SUM(epoch_reward)epoch_reward, SUM(foundation_bonus)foundation_bonus, SUM(priority_bonus)priority_bonus "+
		"FROM block_rewards WHERE epoch_number = ? GROUP BY epoch_number, reward_address", lastEpoch).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		// The chain granted nothing anywhere in this epoch -- possible when the
		// rewarding fund is empty. Leave whatever is already stored alone
		// rather than deleting it in exchange for nothing.
		log.L().Warn("hermes: no block_rewards rows for epoch, nothing to aggregate",
			zap.Uint64("epoch", lastEpoch))
		return nil
	}
	if err := tx.Where("epoch_number = ?", lastEpoch).Delete(&models.HermesAccountReward{}).Error; err != nil {
		return err
	}
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
		if row.PriorityBonus != "" {
			totalPriorityBonus, ok := big.NewInt(0).SetString(row.PriorityBonus, 10)
			if !ok {
				return ErrConvertBigIntString
			}
			totalBlockReward.Add(totalBlockReward, totalPriorityBonus)
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
			if err := tx.Create(&modelHAR).Error; err != nil {
				return err
			}
			continue
		}
		candidateRewardsMap, err := breakdownRewards(tx, epochNumber, candidateNames, weightedVotesMapping,
			totalBlockReward, totalEpochReward, totalFoundationBonus)
		if err != nil {
			return errors.Wrap(err, "failed to get candidate rewards map")
		}
		for candidateName, rewards := range candidateRewardsMap {
			modelHAR := models.HermesAccountReward{
				EpochNumber:     lastEpoch,
				CandidateName:   candidateName,
				BlockReward:     decimal.NewFromBigInt(rewards[0], 0),
				EpochReward:     decimal.NewFromBigInt(rewards[1], 0),
				FoundationBonus: decimal.NewFromBigInt(rewards[2], 0),
			}
			if err := tx.Create(&modelHAR).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func getVotingInfo(tx *gorm.DB, lastEpoch uint64) (map[string][]string, map[string]*big.Int, error) {
	var rows []models.HermesVotingResult
	if err := tx.Model(models.HermesVotingResult{}).Where("epoch_number=?", lastEpoch).Find(&rows).Error; err != nil {
		return nil, nil, err
	}

	if len(rows) == 0 {
		return nil, nil, ErrEmptyRecords
	}
	rewardAddrToNameMapping := make(map[string][]string)
	weightedVotesMapping := make(map[string]*big.Int)
	for _, row := range rows {
		if _, ok := rewardAddrToNameMapping[row.RewardAddress]; !ok {
			rewardAddrToNameMapping[row.RewardAddress] = make([]string, 0)
		}
		rewardAddrToNameMapping[row.RewardAddress] = append(rewardAddrToNameMapping[row.RewardAddress], row.DelegateName)

		totalWeightedVotes := row.TotalWeightedVotes.BigInt()
		weightedVotesMapping[row.DelegateName] = totalWeightedVotes
	}
	return rewardAddrToNameMapping, weightedVotesMapping, nil
}

func breakdownRewards(
	tx *gorm.DB,
	epochNumber uint64,
	candidateNames []string,
	weightedVotesMap map[string]*big.Int,
	totalBlockReward *big.Int,
	totalEpochReward *big.Int,
	totalFoundationBonus *big.Int,
) (map[string][]*big.Int, error) {
	candidateVoteList := make([]*CandidateVote, 0, len(weightedVotesMap))
	for name, votes := range weightedVotesMap {
		candidateVoteList = append(candidateVoteList, &CandidateVote{
			CandidateName:      name,
			TotalWeightedVotes: votes,
		})
	}
	// Sort list by votes in decreasing order
	sort.Slice(candidateVoteList, func(i, j int) bool {
		return candidateVoteList[i].TotalWeightedVotes.Cmp(candidateVoteList[j].TotalWeightedVotes) == 1
	})
	candidateRank := make(map[string]uint64)
	for i, candidateVote := range candidateVoteList {
		candidateRank[candidateVote.CandidateName] = uint64(i + 1)
	}
	productivityMap, err := getProductivity(tx, epochNumber)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get productivity map")
	}
	productionSum := big.NewInt(0)
	qualifiedTotalVotes := big.NewInt(0)
	foundationBonusCount := big.NewInt(0)
	earnBlockReward := make(map[string]bool)
	earnEpochReward := make(map[string]bool)
	earnFoundationBonus := make(map[string]bool)
	for _, candidateName := range candidateNames {
		productive := true
		if productivity, ok := productivityMap[candidateName]; ok {
			productionSum.Add(productionSum, big.NewInt(int64(productivity.Production)))
			earnBlockReward[candidateName] = true
			if productivity.Production*100/productivity.ExpectedProduction < 85 {
				productive = false
			}
		}
		NumDelegatesForEpochReward := uint64(100)
		NumDelegatesForFoundationBonus := uint64(36)
		// qualify for epoch reward
		if candidateRank[candidateName] <= NumDelegatesForEpochReward && productive {
			qualifiedTotalVotes.Add(qualifiedTotalVotes, weightedVotesMap[candidateName])
			earnEpochReward[candidateName] = true
		}
		// qualify for foundation bonus
		if candidateRank[candidateName] <= NumDelegatesForFoundationBonus {
			foundationBonusCount.Add(foundationBonusCount, big.NewInt(1))
			earnFoundationBonus[candidateName] = true
		}
	}
	candidateRewardsMap := make(map[string][]*big.Int)
	for _, candidateName := range candidateNames {
		blockReward := big.NewInt(0)
		epochReward := big.NewInt(0)
		foundationBonus := big.NewInt(0)
		if productionSum.Sign() > 0 && earnBlockReward[candidateName] {
			production := big.NewInt(0).SetUint64(productivityMap[candidateName].Production)
			blockReward = big.NewInt(0).Div(big.NewInt(0).Mul(totalBlockReward, production), productionSum)
		}
		if qualifiedTotalVotes.Sign() > 0 && earnEpochReward[candidateName] {
			epochReward = big.NewInt(0).Div(big.NewInt(0).Mul(totalEpochReward, weightedVotesMap[candidateName]), qualifiedTotalVotes)
		}
		if totalFoundationBonus.Sign() > 0 && earnFoundationBonus[candidateName] {
			foundationBonus = big.NewInt(0).Div(totalFoundationBonus, foundationBonusCount)
		}

		if blockReward.Sign() == 0 && epochReward.Sign() == 0 && foundationBonus.Sign() == 0 {
			continue
		}
		candidateRewardsMap[candidateName] = []*big.Int{blockReward, epochReward, foundationBonus}
	}
	return candidateRewardsMap, nil
}

func getProductivity(tx *gorm.DB, epochNumber uint64) (map[string]*Productivity, error) {
	var rows []ProductivityHistory
	if err := tx.Raw("SELECT t1.epoch_num, t1.expected_producer_name AS producer_name,COALESCE(production,0) as production, COALESCE(expected_production,0) as expected_production FROM (SELECT epoch_num, expected_producer_name, COUNT(expected_producer_address) AS expected_production FROM block_meta WHERE epoch_num = ? GROUP BY epoch_num, expected_producer_name) AS t1 LEFT JOIN (SELECT epoch_num, producer_name, COUNT(producer_address) AS production FROM block_meta WHERE epoch_num = ? GROUP BY epoch_num, producer_name) AS t2 ON t1.epoch_num = t2.epoch_num AND t1.expected_producer_name=t2.producer_name", epochNumber, epochNumber).Find(&rows).Error; err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, errors.Wrapf(errors.New("empty records"), "epoch = %d", epochNumber)
	}

	productivityMap := make(map[string]*Productivity)
	for _, row := range rows {
		productivityMap[row.ProducerName] = &Productivity{
			Production:         row.Production,
			ExpectedProduction: row.ExpectedProduction,
		}
	}
	return productivityMap, nil
}
