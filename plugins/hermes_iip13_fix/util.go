package main

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	"github.com/iotexproject/iotex-election/util"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const (
	// PollProtocolID is ID of poll protocol
	PollProtocolID      = "poll"
	protocolID          = "staking"
	readBucketsLimit    = 300000
	readCandidatesLimit = 20000
)

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

func ownerAddressToNameMap(candidates *iotextypes.CandidateListV2) (ret map[string]string, err error) {
	ret = make(map[string]string)
	for _, can := range candidates.Candidates {
		ret[can.OwnerAddress] = can.Name
	}
	return
}

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

var (
	ErrEmptyRecords        = errors.New("empty records")
	ErrConvertBigIntString = errors.New("failed to convert string to big int")
)
var GenesisVoteWeightCalConsts = genesis.VoteWeightCalConsts{
	DurationLg: 1.2,
	AutoStake:  1,
	SelfStake:  1.06,
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

func selfStakeIndexMap(candidates *iotextypes.CandidateListV2) map[uint64]struct{} {
	ret := make(map[uint64]struct{})
	for _, can := range candidates.Candidates {
		ret[can.SelfStakeBucketIdx] = struct{}{}
	}
	return ret
}

func getLSDStakingBuckets(epochHeight uint64) (results []*SystemStakingBucket, err error) {
	db := db.DB()
	query := `WITH max_ids AS (
		SELECT MAX(id) AS max_id
		FROM system_staking_buckets
		WHERE block_height <= ?
		GROUP BY bucket_id
	)
	SELECT bucket_id,owner_address,delegate_owner_address,staked_amount,auto_stake,duration
	FROM system_staking_buckets t1
	RIGHT JOIN max_ids t2 ON  t1.id=t2.max_id order by bucket_id`
	rows, err := db.Raw(query, epochHeight).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		av := new(SystemStakingBucket)

		if err := db.ScanRows(rows, av); err != nil {
			return nil, err
		}
		results = append(results, av)
	}
	return results, nil
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

// getStakingBuckets get specific buckets by height
func getStakingBuckets(chainClient iotexapi.APIServiceClient, offset, limit uint32, height uint64) (voteBucketList *iotextypes.VoteBucketList, err error) {
	methodName, err := proto.Marshal(&iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_BUCKETS,
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
