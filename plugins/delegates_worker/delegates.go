package main

import (
	"context"
	"math/big"
	"strconv"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/action/protocol/vote"
	"github.com/iotexproject/iotex-core/config"
	"github.com/iotexproject/iotex-core/state"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const (
	protocolID          = "staking"
	readCandidatesLimit = 20000
	defaultDelegateNum  = 36
)

type delegate struct {
	Address            string   `json:"address"`
	Name               string   `json:"string"`
	Rank               int      `json:"rank"`
	Alias              string   `json:"alias"`
	Active             bool     `json:"active"`
	Production         int      `json:"production"`
	Votes              string   `json:"votes"`
	ProbatedStatus     bool     `json:"probatedStatus"`
	TotalWeightedVotes *big.Int `json:"totalWeightedVotes"`
}

type delegatesMessage struct {
	Epoch       int        `json:"epoch"`
	StartBlock  uint64     `json:"startBlock"`
	TotalBlocks int        `json:"totalBlocks"`
	Delegates   []delegate `json:"delegates"`
}

// GetChainMeta gets block chain metadata
func GetChainMeta() (*iotextypes.ChainMeta, error) {
	chainClient := kernel.ChainClient()
	ctx := context.Background()
	res, err := chainClient.GetChainMeta(ctx, &iotexapi.GetChainMetaRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get chain meta")
	}
	return res.ChainMeta, nil
}

// GetEpochMeta gets blockchain epoch meta
func GetEpochMeta(epochNum uint64) (*iotexapi.GetEpochMetaResponse, error) {
	request := &iotexapi.GetEpochMetaRequest{EpochNumber: epochNum}

	chainClient := kernel.ChainClient()
	ctx := context.Background()

	response, err := chainClient.GetEpochMeta(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to invoke GetEpochMeta api")
	}
	return response, nil
}

// GetBucketList get bucket list
func GetBucketList(
	methodName iotexapi.ReadStakingDataMethod_Name,
	readStakingDataRequest *iotexapi.ReadStakingDataRequest,
) (*iotextypes.VoteBucketList, error) {
	chainClient := kernel.ChainClient()
	ctx := context.Background()
	method := &iotexapi.ReadStakingDataMethod{Method: methodName}
	methodData, err := proto.Marshal(method)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal read staking data method")
	}
	requestData, err := proto.Marshal(readStakingDataRequest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal read staking data request")
	}

	request := &iotexapi.ReadStateRequest{
		ProtocolID: []byte("staking"),
		MethodName: methodData,
		Arguments:  [][]byte{requestData},
	}

	response, err := chainClient.ReadState(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to invoke ReadState api")
	}
	bucketlist := iotextypes.VoteBucketList{}
	if err := proto.Unmarshal(response.Data, &bucketlist); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}
	return &bucketlist, nil
}

// GetProbationList gets probation list
func GetProbationList(epochNum uint64, epochStartHeight uint64) (*iotexapi.ReadStateResponse, error) {
	chainClient := kernel.ChainClient()
	ctx := context.Background()

	request := &iotexapi.ReadStateRequest{
		ProtocolID: []byte("poll"),
		MethodName: []byte("ProbationListByEpoch"),
		Arguments:  [][]byte{[]byte(strconv.FormatUint(epochNum, 10))},
		Height:     strconv.FormatUint(epochStartHeight, 10),
	}
	response, err := chainClient.ReadState(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to invoke ReadState api")
	}
	return response, nil
}

func getProbationList(epochNum uint64, epochStartHeight uint64) (*vote.ProbationList, error) {
	probationListRes, err := GetProbationList(epochNum, epochStartHeight)
	if err != nil {
		return nil, err
	}
	probationList := &vote.ProbationList{}
	if probationListRes != nil {
		if err := probationList.Deserialize(probationListRes.Data); err != nil {
			return nil, err
		}
	}
	return probationList, nil
}

func delegates() error {
	epochNum := uint64(0)
	chainMeta, err := GetChainMeta()
	if err != nil {
		return err
	}
	epochData := chainMeta.GetEpoch()
	if epochData == nil {
		return errors.Wrap(err, "ROLLDPOS is not registered")
	}
	epochNum = epochData.Num
	response, err := GetEpochMeta(epochNum)
	if err != nil {
		return err
	}
	epochData = response.EpochData
	message := delegatesMessage{
		Epoch:       int(epochData.Num),
		StartBlock:  epochData.Height,
		TotalBlocks: int(response.TotalBlocks),
	}
	probationList, err := getProbationList(epochNum, epochData.Height)
	if err != nil {
		return errors.Wrap(err, "failed to get probation list")
	}
	if epochData.Height >= config.Default.Genesis.FairbankBlockHeight {
		return delegatesV2(probationList, response, &message)
	}
	for rank, bp := range response.BlockProducersInfo {
		votes, ok := big.NewInt(0).SetString(bp.Votes, 10)
		if !ok {
			return errors.Wrap(err, "failed to convert votes into big int")
		}
		isProbated := false
		if _, ok := probationList.ProbationInfo[bp.Address]; ok {
			// if it exists in probation info
			isProbated = true
		}
		delegate := delegate{
			Address:        bp.Address,
			Rank:           rank + 1,
			Active:         bp.Active,
			Production:     int(bp.Production),
			Votes:          votes.String(),
			ProbatedStatus: isProbated,
		}
		message.Delegates = append(message.Delegates, delegate)
	}
	return sortAndUpdate(&message)
}

func delegatesV2(pb *vote.ProbationList, epochMeta *iotexapi.GetEpochMetaResponse, message *delegatesMessage) error {
	chainClient := kernel.ChainClient()
	ctx := context.Background()

	request := &iotexapi.ReadStateRequest{
		ProtocolID: []byte("poll"),
		MethodName: []byte("ActiveBlockProducersByEpoch"),
		Arguments:  [][]byte{[]byte(strconv.FormatUint(epochMeta.EpochData.Num, 10))},
		Height:     strconv.FormatUint(epochMeta.EpochData.Height, 10),
	}
	abpResponse, err := chainClient.ReadState(ctx, request)
	if err != nil {
		return errors.Wrap(err, "failed to invoke ReadState api")
	}
	var ABPs state.CandidateList
	if err := ABPs.Deserialize(abpResponse.Data); err != nil {
		return errors.Wrap(err, "failed to deserialize active BPs")
	}
	request = &iotexapi.ReadStateRequest{
		ProtocolID: []byte("poll"),
		MethodName: []byte("BlockProducersByEpoch"),
		Arguments:  [][]byte{[]byte(strconv.FormatUint(epochMeta.EpochData.Num, 10))},
	}
	bpResponse, err := chainClient.ReadState(ctx, request)
	if err != nil {
		return errors.Wrap(err, "failed to invoke ReadState api")
	}
	var BPs state.CandidateList
	if err := BPs.Deserialize(bpResponse.Data); err != nil {
		return errors.Wrap(err, "failed to deserialize BPs")
	}
	isActive := make(map[string]bool)
	for _, abp := range ABPs {
		isActive[abp.Address] = true
	}
	production := make(map[string]int)
	for _, info := range epochMeta.BlockProducersInfo {
		production[info.Address] = int(info.Production)
	}
	for rank, bp := range BPs {
		isProbated := false
		if _, ok := pb.ProbationInfo[bp.Address]; ok {
			isProbated = true
		}
		message.Delegates = append(message.Delegates, delegate{
			Address:        bp.Address,
			Rank:           rank + 1,
			Active:         isActive[bp.Address],
			Production:     production[bp.Address],
			Votes:          bp.Votes.String(),
			ProbatedStatus: isProbated,
		})
	}
	fillMessage(chainClient, message, isActive, pb)
	return sortAndUpdate(message)
}

func fillMessage(cli iotexapi.APIServiceClient, message *delegatesMessage, active map[string]bool, pb *vote.ProbationList) error {
	cl, err := getAllStakingCandidates(cli)
	if err != nil {
		return err
	}
	addressMap := make(map[string]*iotextypes.CandidateV2)
	for _, candidate := range cl.Candidates {
		addressMap[candidate.OperatorAddress] = candidate
	}
	delegateAddressMap := make(map[string]struct{})
	for _, m := range message.Delegates {
		delegateAddressMap[m.Address] = struct{}{}
	}
	for i, m := range message.Delegates {
		if c, ok := addressMap[m.Address]; ok {
			message.Delegates[i].Name = c.Name
			continue
		}
	}
	rank := len(message.Delegates) + 1
	for _, candidate := range cl.Candidates {
		if _, ok := delegateAddressMap[candidate.OperatorAddress]; ok {
			continue
		}
		isProbated := false
		if _, ok := pb.ProbationInfo[candidate.OwnerAddress]; ok {
			isProbated = true
		}
		message.Delegates = append(message.Delegates, delegate{
			Address:        candidate.OperatorAddress,
			Name:           candidate.Name,
			Rank:           rank,
			Active:         active[candidate.OperatorAddress],
			Votes:          candidate.TotalWeightedVotes,
			ProbatedStatus: isProbated,
		})
		rank++
	}
	return nil
}

func getAllStakingCandidates(chainClient iotexapi.APIServiceClient) (candidateListAll *iotextypes.CandidateListV2, err error) {
	candidateListAll = &iotextypes.CandidateListV2{}
	for i := uint32(0); ; i++ {
		offset := i * readCandidatesLimit
		size := uint32(readCandidatesLimit)
		candidateList, err := getStakingCandidates(chainClient, offset, size)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get candidates")
		}
		candidateListAll.Candidates = append(candidateListAll.Candidates, candidateList.Candidates...)
		if len(candidateList.Candidates) < readCandidatesLimit {
			break
		}
	}
	return
}

func getStakingCandidates(chainClient iotexapi.APIServiceClient, offset, limit uint32) (candidateList *iotextypes.CandidateListV2, err error) {
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
	}
	readStateRes, err := chainClient.ReadState(context.Background(), readStateRequest)
	if err != nil {
		return
	}
	candidateList = &iotextypes.CandidateListV2{}
	if err := proto.Unmarshal(readStateRes.GetData(), candidateList); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal VoteBucketList")
	}
	return
}

func sortAndUpdate(message *delegatesMessage) error {
	var count int64
	err := db.DB().Model(&NodeDelegates{}).Where("block_height = ?", message.StartBlock).Count(&count).Error
	if err != nil {
		return err
	}
	//skipping exist
	if count > 0 {
		return nil
	}
	return db.DB().Transaction(func(tx *gorm.DB) error {
		for _, bp := range message.Delegates {
			votes, err := decimal.NewFromString(bp.Votes)
			if err != nil {
				return err
			}
			nd := &NodeDelegates{
				BlockHeight:     message.StartBlock,
				ProducerAddress: bp.Address,
				ProducerName:    bp.Name,
				Active:          bp.Active,
				Rank:            bp.Rank,
				Blocks:          bp.Production,
				Probated:        bp.ProbatedStatus,
				Votes:           votes,
			}
			if err := tx.Create(nd).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
