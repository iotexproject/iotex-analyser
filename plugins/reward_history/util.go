package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/action/protocol/rewarding/rewardingpb"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const (
	// PollProtocolID is ID of poll protocol
	PollProtocolID      = "poll"
	protocolID          = "staking"
	readBucketsLimit    = 30000
	readCandidatesLimit = 20000
)

//https://github.com/iotexproject/iotex-analytics/blob/4130adda652ccd3a8b9ecf2fc5be372651274aed/indexprotocol/protocol.go#L125
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

func handleLogs(logs []*action.Log) (map[string]*RewardInfo, error) {
	rewardInfoMap := make(map[string]*RewardInfo)
	for _, l := range logs {
		rewardLog := &rewardingpb.RewardLog{}
		if err := proto.Unmarshal(l.Data, rewardLog); err != nil {
			return rewardInfoMap, errors.Wrap(err, "failed to unmarshal receipt data into reward log")
		}
		rewards, ok := rewardInfoMap[rewardLog.Addr]
		if !ok {
			rewardInfoMap[rewardLog.Addr] = &RewardInfo{
				RewardHistory:   big.NewInt(0),
				EpochReward:     big.NewInt(0),
				FoundationBonus: big.NewInt(0),
			}
			rewards = rewardInfoMap[rewardLog.Addr]
		}
		amount, ok := big.NewInt(0).SetString(rewardLog.Amount, 10)
		if !ok {
			return rewardInfoMap, errors.New("failed to convert reward amount from string to big int")
		}
		switch rewardLog.Type {
		case rewardingpb.RewardLog_BLOCK_REWARD:
			rewards.RewardHistory.Add(rewards.RewardHistory, amount)
		case rewardingpb.RewardLog_EPOCH_REWARD:
			rewards.EpochReward.Add(rewards.EpochReward, amount)
		case rewardingpb.RewardLog_FOUNDATION_BONUS:
			rewards.FoundationBonus.Add(rewards.FoundationBonus, amount)
		default:
			return rewardInfoMap, errors.New("Unknown type of reward")
		}
	}
	return rewardInfoMap, nil
}
