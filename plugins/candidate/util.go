package main

import (
	"context"
	"fmt"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const (
	protocolID       = "staking"
	readBucketsLimit = 300000
)

func GetStakingBucketByID(bucketID, blkHeight uint64) (*iotextypes.VoteBucket, error) {
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	chainClient := kernel.ChainClient()

	for i := uint32(0); ; i++ {
		offset := i * readBucketsLimit
		size := uint32(readBucketsLimit)
		voteBucketList, err := getStakingBuckets(chainClient, offset, size, epochHeight)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get bucket")
		}
		for _, bucket := range voteBucketList.Buckets {
			if bucket.Index == bucketID {
				return bucket, nil
			}
		}
		if len(voteBucketList.Buckets) < readBucketsLimit {
			break
		}
	}
	return nil, nil
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

func fetchCandidateByIDAt(stash map[string]*models.Candidate, id string, height uint64) (*models.Candidate, error) {
	if c, ok := stash[id]; ok {
		return c, nil
	}
	c := &models.Candidate{}
	err := c.FetchByCandidateIDWithHeight(id, height)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get candidate by id")
	}
	return c, nil
}

func fetchCandidateByOwnerAt(stash map[string]*models.Candidate, owner string, height uint64) (*models.Candidate, error) {
	if c, ok := stash[owner]; ok {
		return c, nil
	}
	c := &models.Candidate{}
	err := c.FetchByOwnerAddressWithHeight(owner, height)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get candidate by owner")
	}
	return c, nil
}
