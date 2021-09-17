package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	etypes "github.com/iotexproject/iotex-election/types"
	"github.com/iotexproject/iotex-election/util"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// DelegateProfileABI is the input ABI used to generate the binding from.
const DelegateProfileABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_delegate\",\"type\":\"address\"},{\"name\":\"_name\",\"type\":\"string\"},{\"name\":\"_value\",\"type\":\"bytes\"}],\"name\":\"updateProfileForDelegate\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"register\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_delegate\",\"type\":\"address\"}],\"name\":\"getEncodedProfile\",\"outputs\":[{\"name\":\"code_\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_address\",\"type\":\"address\"}],\"name\":\"isOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"getFieldByName\",\"outputs\":[{\"name\":\"verifier_\",\"type\":\"address\"},{\"name\":\"deprecated_\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_byteCode\",\"type\":\"bytes\"}],\"name\":\"updateProfileWithByteCode\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_name\",\"type\":\"string\"},{\"name\":\"_verifierAddr\",\"type\":\"address\"}],\"name\":\"newField\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_name\",\"type\":\"string\"},{\"name\":\"_value\",\"type\":\"bytes\"}],\"name\":\"updateProfile\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_idx\",\"type\":\"uint256\"}],\"name\":\"getFieldByIndex\",\"outputs\":[{\"name\":\"name_\",\"type\":\"string\"},{\"name\":\"verifier_\",\"type\":\"address\"},{\"name\":\"deprecated_\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_delegate\",\"type\":\"address\"},{\"name\":\"_byteCode\",\"type\":\"bytes\"}],\"name\":\"updateProfileWithByteCodeForDelegate\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"fieldNames\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_addr\",\"type\":\"address\"}],\"name\":\"registered\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_delegate\",\"type\":\"address\"},{\"name\":\"_field\",\"type\":\"string\"}],\"name\":\"getProfileByField\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"deprecateField\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"numOfFields\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"registerAddr\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"FeeUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"delegate\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"name\",\"type\":\"string\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"bytes\"}],\"name\":\"ProfileUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"name\",\"type\":\"string\"}],\"name\":\"FieldDeprecated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"name\",\"type\":\"string\"}],\"name\":\"NewField\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Pause\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Unpause\",\"type\":\"event\"}]"
const (
	topicProfileUpdated     = "217aa5ef0b78f028d51fd573433bdbe2daf6f8505e6a71f3af1393c8440b341b"
	blockRewardPortion      = "blockRewardPortion"
	epochRewardPortion      = "epochRewardPortion"
	foundationRewardPortion = "foundationRewardPortion"
	RewardPortionContract   = "io1lfl4ppn2c3wcft04f0rk0jy9lyn4pcjcm7638u"
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

func getDelegateNameFromTopic(logTopic hash.Hash256) string {
	n := bytes.IndexByte(logTopic[:], 0)
	return string(logTopic[:n])
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
		Height:     fmt.Sprintf("%d", height),
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
	delegateABI, err := abi.JSON(strings.NewReader(DelegateProfileABI))
	if err != nil {
		err = errors.Wrap(err, "failed to get parsed delegate profile ABI interface")
		return
	}

	// get from mysql first
	blockRewardPercentage, epochRewardPercentage, foundationBonusPercentage, err = getLastEpochPortion(epochNumber - 1)
	if err != nil {
		err = errors.Wrap(err, "failed to get last epoch portion")
		return
	}

	//and then update from contract from last epochstartHeight to this epochStartheight-1
	lastEpochStartHeight := kernel.GetEpochHeight(epochNumber - 1)
	if epochStartHeight < lastEpochStartHeight {
		err = errors.Wrap(err, "epoch start height less than last epoch start height")
		return
	}
	count := epochStartHeight - lastEpochStartHeight
	var blockRewardFromLog, epochRewardFromLog, foundationBonusFromLog map[string]float64
	blockRewardFromLog, epochRewardFromLog, foundationBonusFromLog, err = getLog(RewardPortionContract, lastEpochStartHeight, count, chainClient, delegateABI)
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

func getLastEpochPortion(epochNumber uint64) (blockReward, epochReward, foundationReward map[string]float64, err error) {
	blockReward = make(map[string]float64)
	epochReward = make(map[string]float64)
	foundationReward = make(map[string]float64)

	var res []models.HermesVotingResult
	db := db.DB()
	result := db.Model(&models.HermesVotingResult{}).Where("epoch_number=?", epochNumber).Find(&res)
	if result.Error != nil {
		err = result.Error
		return
	}

	if len(res) == 0 {
		err = errors.New("record empty")
		return
	}

	for _, vr := range res {
		brp, _ := vr.BlockRewardPercentage.Float64()
		blockReward[vr.StakingAddress] = brp
		erp, _ := vr.EpochRewardPercentage.Float64()
		epochReward[vr.StakingAddress] = erp
		fbp, _ := vr.FoundationBonusPercentage.Float64()
		foundationReward[vr.StakingAddress] = fbp
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

				evt, err := delegateProfileABI.Unpack("ProfileUpdated", l.Data)
				if err != nil {
					continue
				}
				fmt.Printf("%v", evt)
				// event := evt.(*DelegateProfileProfileUpdated)
				// switch event.Name {
				// case blockRewardPortion:
				// 	blockReward[hex.EncodeToString(event.Delegate.Bytes())] = float64(big.NewInt(0).SetBytes(event.Value).Uint64()) / 100
				// case epochRewardPortion:
				// 	epochReward[hex.EncodeToString(event.Delegate.Bytes())] = float64(big.NewInt(0).SetBytes(event.Value).Uint64()) / 100
				// case foundationRewardPortion:
				// 	foundationReward[hex.EncodeToString(event.Delegate.Bytes())] = float64(big.NewInt(0).SetBytes(event.Value).Uint64()) / 100
				// }
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
