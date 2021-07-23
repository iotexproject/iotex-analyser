package main

import (
	"context"
	"strconv"

	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

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
