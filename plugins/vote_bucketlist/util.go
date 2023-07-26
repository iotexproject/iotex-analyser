package main

import (
	"context"
	"fmt"

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

// GetAllStakingBuckets get all buckets by height
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
