package voter_reward

import (
	"context"
	"fmt"
	"math/big"

	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/rewardingpb"
	"github.com/iotexproject/iotex-core/v2/action/protocol/staking/stakingpb"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const rewardingProtocolID = "rewarding"

// ErrNoSnapshot reports that a delegate has no frozen snapshot for the era
// being read, which is how the protocol says "this delegate was not opted in
// when the era froze".
var ErrNoSnapshot = errors.New("voter_reward: delegate has no reward snapshot")

// ReadDistributionState returns the settlement plan and progress as of height.
//
// Everything the era summary needs arrives in this one call: the freeze
// height, the per-delegate frozen and distributed amounts, and the scan
// cursor. Reconstructing those from receipt logs alone is not possible — the
// CURSOR_PROGRESS log carries the cursor but not the plan.
func ReadDistributionState(
	ctx context.Context, client iotexapi.APIServiceClient, height uint64,
) (*rewardingpb.VoterRewardDistributionState, error) {
	res, err := client.ReadState(ctx, &iotexapi.ReadStateRequest{
		ProtocolID: []byte(rewardingProtocolID),
		MethodName: []byte("VoterRewardDistribution"),
		Height:     fmt.Sprintf("%d", height),
	})
	if err != nil {
		return nil, errors.Wrap(err, "read VoterRewardDistribution")
	}
	state := &rewardingpb.VoterRewardDistributionState{}
	if err := proto.Unmarshal(res.GetData(), state); err != nil {
		return nil, errors.Wrap(err, "unmarshal VoterRewardDistributionState")
	}
	return state, nil
}

// DelegateSnapshot is the frozen per-delegate reward policy for one era.
type DelegateSnapshot struct {
	BlockCommissionBps   uint64
	EpochCommissionBps   uint64
	CommissionConfigured bool
	TotalWeight          *big.Int
	FreezeHeight         uint64
	SelfStakeBucketIdx   uint64
}

// ReadDelegateSnapshot returns the frozen policy for one delegate, or
// ErrNoSnapshot when the delegate was not opted in at the freeze.
func ReadDelegateSnapshot(
	ctx context.Context, client iotexapi.APIServiceClient, delegateID string, height uint64,
) (*DelegateSnapshot, error) {
	res, err := client.ReadState(ctx, &iotexapi.ReadStateRequest{
		ProtocolID: []byte(rewardingProtocolID),
		MethodName: []byte("DelegateRewardSnapshot"),
		Arguments:  [][]byte{[]byte(delegateID)},
		Height:     fmt.Sprintf("%d", height),
	})
	if err != nil {
		// The protocol surfaces "not opted in at the freeze" as a state-not-
		// exist error rather than an empty record, so this is an expected
		// outcome and not an infrastructure failure.
		return nil, errors.Wrapf(ErrNoSnapshot, "%s: %v", delegateID, err)
	}
	snap := &stakingpb.CandidateRewardSnapshot{}
	if err := proto.Unmarshal(res.GetData(), snap); err != nil {
		return nil, errors.Wrap(err, "unmarshal CandidateRewardSnapshot")
	}
	return &DelegateSnapshot{
		BlockCommissionBps:   snap.GetBlockCommissionBasisPoints(),
		EpochCommissionBps:   snap.GetEpochCommissionBasisPoints(),
		CommissionConfigured: snap.GetCommissionConfigured(),
		TotalWeight:          new(big.Int).SetBytes(snap.GetTotalWeight()),
		FreezeHeight:         snap.GetFreezeHeight(),
		SelfStakeBucketIdx:   snap.GetSelfStakeBucketIdx(),
	}, nil
}

// PayoutRouting is where a delegate's own commission goes, and whether IIP-59
// is routing its voter rewards on chain.
type PayoutRouting struct {
	Address              string
	OnchainRewardEnabled bool
}

// ReadPayoutAddress is the authoritative answer to "is this delegate opted
// in", and the only one that catches the Hermes migration at the fork block —
// that migration flips the opt-in bit inside CreatePreStates and emits no
// action and no receipt log, so an event-only indexer never sees it.
func ReadPayoutAddress(
	ctx context.Context, client iotexapi.APIServiceClient, delegateID string, height uint64,
) (*PayoutRouting, error) {
	res, err := client.ReadState(ctx, &iotexapi.ReadStateRequest{
		ProtocolID: []byte(rewardingProtocolID),
		MethodName: []byte("DelegatePayoutAddress"),
		Arguments:  [][]byte{[]byte(delegateID)},
		Height:     fmt.Sprintf("%d", height),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "read DelegatePayoutAddress %s", delegateID)
	}
	out := &rewardingpb.DelegatePayoutAddress{}
	if err := proto.Unmarshal(res.GetData(), out); err != nil {
		return nil, errors.Wrap(err, "unmarshal DelegatePayoutAddress")
	}
	addr, err := addressString(out.GetAddress())
	if err != nil {
		return nil, err
	}
	return &PayoutRouting{Address: addr, OnchainRewardEnabled: out.GetOnchainRewardEnabled()}, nil
}
