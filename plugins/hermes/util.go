package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"slices"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/service/hermes_reward"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
	etypes "github.com/iotexproject/iotex-election/types"
	"github.com/iotexproject/iotex-election/util"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	topicProfileUpdated                      = "217aa5ef0b78f028d51fd573433bdbe2daf6f8505e6a71f3af1393c8440b341b"
	blockRewardPortion                       = "blockRewardPortion"
	epochRewardPortion                       = "epochRewardPortion"
	foundationRewardPortion                  = "foundationRewardPortion"
	RewardPortionContract                    = "io1lfl4ppn2c3wcft04f0rk0jy9lyn4pcjcm7638u"
	RewardportionContractDeployHeight        = 5095225
	maxBlockRange                     uint64 = 1000000
)
const (
	// PollProtocolID is ID of poll protocol
	PollProtocolID      = "poll"
	protocolID          = "staking"
	readBucketsLimit    = 300000
	readCandidatesLimit = 20000
)

var GenesisVoteWeightCalConsts = genesis.VoteWeightCalConsts{
	DurationLg: 1.2,
	AutoStake:  1,
	SelfStake:  1.06,
}

// ErrConvertBigIntString aliases the sentinel in service/hermes_reward so a
// caller matching on errors from either package sees the same value.
var ErrConvertBigIntString = hermes_reward.ErrConvertBigIntString

func getDelegateNameFromTopic(logTopic hash.Hash256) string {
	n := bytes.IndexByte(logTopic[:], 0)
	return string(logTopic[:n])
}

func GetAllStakingBuckets(chainClient iotexapi.APIServiceClient, epochHeight uint64) (voteBucketListAll *iotextypes.VoteBucketList, err error) {
	voteBucketListAll = &iotextypes.VoteBucketList{}
	for i := uint32(0); ; i++ {
		offset := i * readBucketsLimit
		size := uint32(readBucketsLimit)
		voteBucketList, err := getStakingBuckets(chainClient, offset, size, epochHeight)
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
		Method: iotexapi.ReadStakingDataMethod_COMPOSITE_BUCKETS,
	})
	if err != nil {
		return nil, err
	}
	arguments := &iotexapi.ReadStakingDataRequest{
		Request: &iotexapi.ReadStakingDataRequest_Buckets{
			Buckets: &iotexapi.ReadStakingDataRequest_VoteBuckets{
				Pagination: &iotexapi.PaginationParam{
					Offset: offset,
					Limit:  limit,
				},
			},
		},
	}
	argumentsBytes, _ := proto.Marshal(arguments)
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte(protocolID),
		MethodName: methodName,
		Arguments:  [][]byte{argumentsBytes},
		Height:     fmt.Sprintf("%d", height),
	}
	ctx := context.WithValue(context.Background(), &iotexapi.ReadStateRequest{}, iotexapi.ReadStakingDataMethod_COMPOSITE_BUCKETS)
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

func getLogFromDB(contractAddress string, topicsFilter [][]byte, from, count uint64) ([]*iotextypes.Log, error) {
	var receiptLogs []*models.BlockReceiptLog
	err := db.DB().Model(&models.BlockReceiptLog{}).Where("address = ? AND block_height >= ? AND block_height < ?", contractAddress, from, from+count).Order("block_height, tx_index, index, id").Find(&receiptLogs).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logs from db")
	}
	logs := make([]*iotextypes.Log, 0, len(receiptLogs))
	for _, receiptLog := range receiptLogs {
		topics := [][]byte{}
		want := len(topicsFilter) <= 0
		for _, topic := range []string{receiptLog.Topic0, receiptLog.Topic1, receiptLog.Topic2, receiptLog.Topic3} {
			if topic != "" {
				t, err := hex.DecodeString(topic)
				if err != nil {
					return nil, errors.Wrap(err, "failed to decode topic")
				}
				topics = append(topics, t)
				// filter out topics
				if len(topicsFilter) > 0 {
					if slices.ContainsFunc(topicsFilter, func(e []byte) bool {
						return bytes.Equal(e, t)
					}) {
						want = true
					}
				}
			}
		}

		if !want {
			continue
		}
		hash, err := hex.DecodeString(receiptLog.ActionHash)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode action hash")
		}
		log := &iotextypes.Log{
			ContractAddress: receiptLog.Address,
			Topics:          topics,
			Data:            receiptLog.Data,
			Index:           uint32(receiptLog.Index),
			TxIndex:         uint32(receiptLog.TxIndex),
			BlkHeight:       receiptLog.BlockHeight,
			ActHash:         hash,
		}
		logs = append(logs, log)
	}
	return logs, nil
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

	shard := count / maxBlockRange
	if count%maxBlockRange != 0 {
		shard++
	}

	for i := uint64(0); i < shard; i++ {
		shardCount := maxBlockRange
		if i == shard-1 && count%maxBlockRange != 0 {
			shardCount = count % maxBlockRange
		}

		logs, err := getLogFromDB(contractAddress, topics, from+i*maxBlockRange, shardCount)
		if err != nil {
			return nil, nil, nil, err
		}

		for _, l := range logs {
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
	}

	return
}

func ownerAddressToNameMap(candidates *iotextypes.CandidateListV2) (ret map[string]string, err error) {
	ret = make(map[string]string)
	for _, can := range candidates.Candidates {
		ret[can.Id] = can.Name
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
					probationMap[can.Id] = pb.Count
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
		return nil, ErrConvertBigIntString
	}
	amount := new(big.Float).SetInt(amountInt)
	weightedAmount, _ := amount.Mul(amount, big.NewFloat(weight)).Int(nil)
	return weightedAmount, nil
}
