package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	etypes "github.com/iotexproject/iotex-election/types"
	"github.com/iotexproject/iotex-election/util"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const (
	topicProfileUpdated               = "217aa5ef0b78f028d51fd573433bdbe2daf6f8505e6a71f3af1393c8440b341b"
	blockRewardPortion                = "blockRewardPortion"
	epochRewardPortion                = "epochRewardPortion"
	foundationRewardPortion           = "foundationRewardPortion"
	RewardPortionContract             = "io1lfl4ppn2c3wcft04f0rk0jy9lyn4pcjcm7638u"
	RewardportionContractDeployHeight = 5095225
)
const (
	// PollProtocolID is ID of poll protocol
	PollProtocolID      = "poll"
	protocolID          = "staking"
	readBucketsLimit    = 30000
	readCandidatesLimit = 20000
)

var GenesisVoteWeightCalConsts = genesis.VoteWeightCalConsts{
	DurationLg: 1.2,
	AutoStake:  1,
	SelfStake:  1.06,
}
var (
	ErrEmptyRecords = errors.New("empty records")
)

type AggregateReward struct {
	EpochNumber     uint64
	RewardAddress   string
	BlockReward     string
	EpochReward     string
	FoundationBonus string
}

type (
	CandidateVote struct {
		CandidateName      string
		TotalWeightedVotes *big.Int
	}
	Productivity struct {
		Production uint64
	}
	ProductivityHistory struct {
		EpochNumber  uint64
		ProducerName string
		Production   uint64
	}
)

func getDelegateNameFromTopic(logTopic hash.Hash256) string {
	n := bytes.IndexByte(logTopic[:], 0)
	return string(logTopic[:n])
}

func getVotingInfo(lastEpoch uint64) (map[string][]string, map[string]*big.Int, error) {

	db := db.DB()
	var rows []models.HermesVotingResult
	if err := db.Model(models.HermesVotingResult{}).Where("epoch_number=?", lastEpoch).Find(&rows).Error; err != nil {
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
		// totalWeightedVotes, err := big.NewInt(0).SetString(row.TotalWeightedVotes, 10)
		// if err != nil {
		// 	return nil, nil, errors.Wrap(err, "failed to covert string to big int")
		// }
		weightedVotesMapping[row.DelegateName] = totalWeightedVotes
	}
	return rewardAddrToNameMapping, weightedVotesMapping, nil
}

func rebuildAccountRewardTable(lastEpoch uint64) error {
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

	if len(rows) > 0 {
		err := db.Where("epoch_number = ?", lastEpoch).Delete(&models.HermesAccountReward{}).Error
		if err != nil {
			return err
		}
	}
	err = db.Transaction(func(tx *gorm.DB) error {
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
				modelHAR := &models.HermesAccountReward{
					EpochNumber:     lastEpoch,
					CandidateName:   candidateName,
					BlockReward:     decimal.NewFromBigInt(totalBlockReward, 0),
					EpochReward:     decimal.NewFromBigInt(totalEpochReward, 0),
					FoundationBonus: decimal.NewFromBigInt(totalFoundationBonus, 0),
				}
				if err := tx.Create(modelHAR).Error; err != nil {
					return err
				}
				continue
			}
			candidateRewardsMap, err := breakdownRewards(epochNumber, candidateNames, weightedVotesMapping,
				totalBlockReward, totalEpochReward, totalFoundationBonus)
			if err != nil {
				return errors.Wrap(err, "failed to get candidate rewards map")
			}
			for candidateName, rewards := range candidateRewardsMap {
				modelHAR := &models.HermesAccountReward{
					EpochNumber:     lastEpoch,
					CandidateName:   candidateName,
					BlockReward:     decimal.NewFromBigInt(rewards[0], 0),
					EpochReward:     decimal.NewFromBigInt(rewards[1], 0),
					FoundationBonus: decimal.NewFromBigInt(rewards[2], 0),
				}
				if err := tx.Create(modelHAR).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	return err
}

func breakdownRewards(
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
	productivityMap, err := getProductivity(epochNumber)
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

func getProductivity(epochNumber uint64) (map[string]*Productivity, error) {

	db := db.DB()
	var rows []ProductivityHistory
	if err := db.Raw("SELECT epoch_num, producer_name, COUNT(producer_address) AS production FROM block_meta WHERE epoch_num = ? GROUP BY epoch_num, producer_name", epochNumber).Find(&rows).Error; err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, errors.New("empty records")
	}

	productivityMap := make(map[string]*Productivity)
	for _, row := range rows {
		productivityMap[row.ProducerName] = &Productivity{
			Production: row.Production,
		}
	}
	return productivityMap, nil
}

// GetAllStakingBuckets get all buckets by height
func GetAllStakingBuckets(chainClient iotexapi.APIServiceClient, height uint64) (voteBucketListAll *iotextypes.VoteBucketList, err error) {
	voteBucketListAll = &iotextypes.VoteBucketList{}
	for i := uint32(0); ; i++ {
		offset := i * readBucketsLimit
		size := uint32(readBucketsLimit)
		voteBucketList, err := getStakingBuckets(chainClient, offset, size, height)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get bucket")
		}
		for _, bucket := range voteBucketList.Buckets {
			if bucket.UnstakeStartTime.AsTime().After(bucket.StakeStartTime.AsTime()) {
				continue
			}
			voteBucketListAll.Buckets = append(voteBucketListAll.Buckets, bucket)
		}
		if len(voteBucketList.Buckets) < readBucketsLimit {
			break
		}
	}
	return
}

// getStakingBuckets get specific buckets by height
func getStakingBuckets(chainClient iotexapi.APIServiceClient, offset, limit uint32, height uint64) (voteBucketList *iotextypes.VoteBucketList, err error) {
	methodName, err := proto.Marshal(&iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_BUCKETS,
	})
	if err != nil {
		return nil, err
	}
	arg, err := proto.Marshal(&iotexapi.ReadStakingDataRequest{
		Request: &iotexapi.ReadStakingDataRequest_Buckets{
			Buckets: &iotexapi.ReadStakingDataRequest_VoteBuckets{
				Pagination: &iotexapi.PaginationParam{
					Offset: offset,
					Limit:  limit,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte(protocolID),
		MethodName: methodName,
		Arguments:  [][]byte{arg},
		Height:     fmt.Sprintf("%d", height),
	}
	ctx := context.WithValue(context.Background(), &iotexapi.ReadStateRequest{}, iotexapi.ReadStakingDataMethod_BUCKETS)
	readStateRes, err := chainClient.ReadState(ctx, readStateRequest)
	if err != nil {
		return
	}
	voteBucketList = &iotextypes.VoteBucketList{}
	if err := proto.Unmarshal(readStateRes.GetData(), voteBucketList); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal VoteBucketList")
	}
	return
}

// GetAllStakingCandidates get all candidates by height
func GetAllStakingCandidates(chainClient iotexapi.APIServiceClient, height uint64) (candidateListAll *iotextypes.CandidateListV2, err error) {
	candidateListAll = &iotextypes.CandidateListV2{}
	for i := uint32(0); ; i++ {
		offset := i * readCandidatesLimit
		size := uint32(readCandidatesLimit)
		candidateList, err := getStakingCandidates(chainClient, offset, size, height)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get candidates")
		}
		// filter out candidates whose master bucket are unstaked/withdrawn
		for _, c := range candidateList.Candidates {
			if c.SelfStakingTokens != "0" {
				candidateListAll.Candidates = append(candidateListAll.Candidates, c)
			}
		}
		if len(candidateList.Candidates) < readCandidatesLimit {
			break
		}
	}
	return
}

// getStakingCandidates get specific candidates by height
func getStakingCandidates(chainClient iotexapi.APIServiceClient, offset, limit uint32, height uint64) (candidateList *iotextypes.CandidateListV2, err error) {
	methodName, err := proto.Marshal(&iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_CANDIDATES,
	})
	if err != nil {
		return nil, err
	}
	arg, err := proto.Marshal(&iotexapi.ReadStakingDataRequest{
		Request: &iotexapi.ReadStakingDataRequest_Candidates_{
			Candidates: &iotexapi.ReadStakingDataRequest_Candidates{
				Pagination: &iotexapi.PaginationParam{
					Offset: offset,
					Limit:  limit,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte(protocolID),
		MethodName: methodName,
		Arguments:  [][]byte{arg},
		Height:     strconv.FormatUint(height, 10),
	}
	ctx := context.WithValue(context.Background(), &iotexapi.ReadStateRequest{}, iotexapi.ReadStakingDataMethod_CANDIDATES)
	readStateRes, err := chainClient.ReadState(ctx, readStateRequest)
	if err != nil {
		return
	}
	candidateList = &iotextypes.CandidateListV2{}
	if err := proto.Unmarshal(readStateRes.GetData(), candidateList); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal VoteBucketList")
	}
	return
}

// filterStakingCandidates returns filtered candidate list by given raw candidate and probation list
func filterStakingCandidates(
	candidates *iotextypes.CandidateListV2,
	unqualifiedList *iotextypes.ProbationCandidateList,
	epochStartHeight uint64,
) (*iotextypes.CandidateListV2, error) {
	candidatesMap := make(map[string]*iotextypes.CandidateV2)
	updatedVotingPower := make(map[string]*big.Int)
	intensityRate := float64(uint32(100)-unqualifiedList.IntensityRate) / float64(100)

	probationMap := make(map[string]uint32)
	for _, elem := range unqualifiedList.ProbationList {
		probationMap[elem.Address] = elem.Count
	}
	for _, cand := range candidates.Candidates {
		filterCand := *cand
		votingPowerInt, ok := new(big.Int).SetString(cand.TotalWeightedVotes, 10)
		if !ok {
			return nil, errors.New("total weighted votes convert error")
		}
		votingPower := new(big.Float).SetInt(votingPowerInt)
		if _, ok := probationMap[cand.OperatorAddress]; ok {
			newVotingPower, _ := votingPower.Mul(votingPower, big.NewFloat(intensityRate)).Int(nil)
			filterCand.TotalWeightedVotes = newVotingPower.String()
		}
		totalWeightedVotes, ok := new(big.Int).SetString(filterCand.TotalWeightedVotes, 10)
		if !ok {
			return nil, errors.New("total weighted votes convert error")
		}
		updatedVotingPower[cand.OperatorAddress] = totalWeightedVotes
		candidatesMap[cand.OperatorAddress] = &filterCand
	}
	// sort again with updated voting power
	sorted := util.Sort(updatedVotingPower, epochStartHeight)
	verifiedCandidates := &iotextypes.CandidateListV2{}
	for _, name := range sorted {
		verifiedCandidates.Candidates = append(verifiedCandidates.Candidates, candidatesMap[name])
	}
	return verifiedCandidates, nil
}

func fetchProbationList(cli iotexapi.APIServiceClient, epochNum uint64) (*iotextypes.ProbationCandidateList, error) {
	request := &iotexapi.ReadStateRequest{
		ProtocolID: []byte("poll"),
		MethodName: []byte("ProbationListByEpoch"),
		Arguments:  [][]byte{[]byte(strconv.FormatUint(epochNum, 10))},
	}
	out, err := cli.ReadState(context.Background(), request)
	if err != nil {
		sta, ok := status.FromError(err)
		if ok && sta.Code() == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	probationList := &iotextypes.ProbationCandidateList{}
	if out.Data != nil {
		if err := proto.Unmarshal(out.Data, probationList); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal probationList")
		}
	}
	return probationList, nil
}

func getAllStakingDelegateRewardPortions(epochStartHeight, epochNumber uint64, chainClient iotexapi.APIServiceClient) (blockRewardPercentage, epochRewardPercentage, foundationBonusPercentage map[string]float64, err error) {
	blockRewardPercentage = make(map[string]float64)
	epochRewardPercentage = make(map[string]float64)
	foundationBonusPercentage = make(map[string]float64)

	// if epochStartHeight == kernel.FairbankEffectiveHeight() {
	// 	count := epochStartHeight - RewardportionContractDeployHeight
	// 	blockRewardPercentage, epochRewardPercentage, foundationBonusPercentage, err = getLog(RewardPortionContract, RewardportionContractDeployHeight, count, chainClient, delegateProfileABI)
	// 	if err != nil {
	// 		err = errors.Wrap(err, "failed to get log from chain")
	// 	}
	// 	return
	// }

	// if len(BlockRewardPercentage) == 0 &&
	// 	len(EpochRewardPercentage) == 0 &&
	// 	len(FoundationBonusPercentage) == 0 {
	// 	count := epochStartHeight - RewardportionContractDeployHeight
	// 	blockRewardPercentage, epochRewardPercentage, foundationBonusPercentage, err = getLog(RewardPortionContract, RewardportionContractDeployHeight, count, chainClient, delegateProfileABI)
	// 	if err != nil {
	// 		err = errors.Wrap(err, "failed to get log from chain")
	// 	}
	// }
	//and then update from contract from last epochstartHeight to this epochStartheight-1

	count := epochStartHeight - RewardportionContractDeployHeight
	var blockRewardFromLog, epochRewardFromLog, foundationBonusFromLog map[string]float64
	blockRewardFromLog, epochRewardFromLog, foundationBonusFromLog, err = getLog(RewardPortionContract, RewardportionContractDeployHeight, count, chainClient, delegateProfileABI)
	if err != nil {
		err = errors.Wrap(err, "failed to get log from chain")
		return
	}
	// update to mysql's portion
	for k, v := range blockRewardFromLog {
		blockRewardPercentage[k] = v
	}
	for k, v := range epochRewardFromLog {
		epochRewardPercentage[k] = v
	}
	for k, v := range foundationBonusFromLog {
		foundationBonusPercentage[k] = v
	}

	return
}

type DelegateProfileProfileUpdated struct {
	Delegate common.Address
	Name     string
	Value    []byte
	Raw      types.Log // Blockchain specific contextual infos
}

func getLog(contractAddress string, from, count uint64, chainClient iotexapi.APIServiceClient, delegateProfileABI abi.ABI) (blockReward, epochReward, foundationReward map[string]float64, err error) {
	blockReward = make(map[string]float64)
	epochReward = make(map[string]float64)
	foundationReward = make(map[string]float64)
	tp, err := hex.DecodeString(topicProfileUpdated)
	if err != nil {
		return
	}
	topics := [][]byte{tp}

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
	if err != nil {
		return
	}
	for _, l := range response.Logs {
		for _, topic := range l.Topics {
			switch hex.EncodeToString(topic) {
			case topicProfileUpdated:
				event := &DelegateProfileProfileUpdated{}
				err := delegateProfileABI.UnpackIntoInterface(event, "ProfileUpdated", l.Data)
				if err != nil {
					continue
				}
				addr, _ := address.FromHex(event.Delegate.String())

				switch event.Name {
				case blockRewardPortion:
					blockReward[addr.String()] = float64(big.NewInt(0).SetBytes(event.Value).Uint64()) / 100
				case epochRewardPortion:
					epochReward[addr.String()] = float64(big.NewInt(0).SetBytes(event.Value).Uint64()) / 100
				case foundationRewardPortion:
					foundationReward[addr.String()] = float64(big.NewInt(0).SetBytes(event.Value).Uint64()) / 100
				}
			}
		}
	}
	return
}

func ownerAddressToNameMap(candidates *iotextypes.CandidateListV2) (ret map[string]string, err error) {
	ret = make(map[string]string)
	for _, can := range candidates.Candidates {
		ret[can.OwnerAddress] = can.Name
	}
	return
}

type (
	// ProbationList defines the schema of "probation_list" table
	ProbationList struct {
		EpochNumber   uint64
		IntensityRate uint64
		Address       string
		Count         uint64
	}

	aggregateKey struct {
		epochNumber   uint64
		candidateName string
		voterAddress  string
		isNative      bool
	}
)

func convertProbationListToLocal(probationList *iotextypes.ProbationCandidateList) (ret []*ProbationList) {
	if probationList == nil {
		return nil
	}
	ret = make([]*ProbationList, 0)
	for _, pb := range probationList.ProbationList {
		p := &ProbationList{
			0,
			uint64(probationList.IntensityRate),
			pb.Address,
			uint64(pb.Count),
		}
		ret = append(ret, p)
	}
	return
}

func stakingProbationListToMap(candidateList *iotextypes.CandidateListV2, probationList []*ProbationList) (intensityRate float64, probationMap map[string]uint64) {
	probationMap = make(map[string]uint64)
	if probationList != nil {
		for _, can := range candidateList.Candidates {
			for _, pb := range probationList {
				intensityRate = float64(uint64(100)-pb.IntensityRate) / float64(100)
				if pb.Address == can.OperatorAddress {
					probationMap[can.OwnerAddress] = pb.Count
				}
			}
		}
	}
	return
}

func probationListToMap(delegates []*etypes.Candidate, pblist []*ProbationList) (intensityRate float64, probationMap map[string]uint64) {
	probationMap = make(map[string]uint64)
	if pblist != nil {
		for _, delegate := range delegates {
			delegateOpAddr := string(delegate.OperatorAddress())
			for _, pb := range pblist {
				intensityRate = float64(uint64(100)-pb.IntensityRate) / float64(100)
				if pb.Address == delegateOpAddr {
					probationMap[hex.EncodeToString(delegate.Name())] = pb.Count
				}
			}
		}
	}
	return
}

func selfStakeIndexMap(candidates *iotextypes.CandidateListV2) map[uint64]struct{} {
	ret := make(map[uint64]struct{})
	for _, can := range candidates.Candidates {
		ret[can.SelfStakeBucketIdx] = struct{}{}
	}
	return ret
}

// CalculateVoteWeight calculates the weighted votes
func CalculateVoteWeight(cfg genesis.VoteWeightCalConsts, v *iotextypes.VoteBucket, selfStake bool) (*big.Int, error) {
	// TODO: calculation of remaining time is wrong
	remainingTime := float64(v.StakedDuration * 86400)
	weight := float64(1)
	var m float64
	if v.AutoStake {
		m = cfg.AutoStake
	}
	if remainingTime > 0 {
		weight += math.Log(math.Ceil(remainingTime/86400)*(1+m)) / math.Log(cfg.DurationLg) / 100
	}
	if selfStake && v.AutoStake && v.StakedDuration >= 91 {
		// self-stake extra bonus requires enable auto-stake for at least 3 months
		weight *= cfg.SelfStake
	}

	amountInt, ok := big.NewInt(0).SetString(v.StakedAmount, 10)
	if !ok {
		return nil, errors.New("failed to convert string to big int")
	}
	amount := new(big.Float).SetInt(amountInt)
	weightedAmount, _ := amount.Mul(amount, big.NewFloat(weight)).Int(nil)
	return weightedAmount, nil
}
