package voter_reward

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/rewardingpb"
	"github.com/iotexproject/iotex-core/v2/action/protocol/staking/stakingpb"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const rewardingProtocolID = "rewarding"

// readStateTimeout bounds a single ReadState round-trip. Without it a hung
// node stalls the indexer indefinitely: these reads sit on the block
// processing path, and snapshotDelegateConfigs makes two of them per candidate
// per era. Matches the 5s readScheduledAtFromChain uses in staking_actions.
const readStateTimeout = 5 * time.Second

// ErrNoSnapshot reports that a delegate has no frozen snapshot for the era
// being read, which is how the protocol says "this delegate was not opted in
// when the era froze".
var ErrNoSnapshot = errors.New("voter_reward: delegate has no reward snapshot")

// isStateNotFound distinguishes "the protocol has no such state" from an
// infrastructure failure.
//
// iotex-core maps every readState miss to codes.NotFound (api/coreservice.go
// ReadState), while a dead connection, an exceeded deadline or an overloaded
// node surface as Unavailable / DeadlineExceeded / ResourceExhausted. Treating
// the latter as "no state" is what silently persists a zero commission for a
// delegate that actually had one — and the era memo means it is never retried.
func isStateNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}

// ReadDistributionState returns the settlement plan and progress as of height.
//
// Everything the era summary needs arrives in this one call: the freeze
// height, the per-delegate frozen and distributed amounts, and the scan
// cursor. Reconstructing those from receipt logs alone is not possible — the
// CURSOR_PROGRESS log carries the cursor but not the plan.
func ReadDistributionState(
	ctx context.Context, client iotexapi.APIServiceClient, height uint64,
) (*rewardingpb.VoterRewardDistributionState, error) {
	ctx, cancel := context.WithTimeout(ctx, readStateTimeout)
	defer cancel()
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
	ctx, cancel := context.WithTimeout(ctx, readStateTimeout)
	defer cancel()
	res, err := client.ReadState(ctx, &iotexapi.ReadStateRequest{
		ProtocolID: []byte(rewardingProtocolID),
		MethodName: []byte("DelegateRewardSnapshot"),
		Arguments:  [][]byte{[]byte(delegateID)},
		Height:     fmt.Sprintf("%d", height),
	})
	if err != nil {
		// The protocol surfaces "not opted in at the freeze" as a state-not-
		// exist error rather than an empty record, so that one is an expected
		// outcome. Everything else is an infrastructure failure and must be
		// propagated: the caller treats ErrNoSnapshot as "zero commission is
		// correct", and the era memo means a wrong zero is never revisited.
		if isStateNotFound(err) {
			return nil, errors.Wrapf(ErrNoSnapshot, "%s: %v", delegateID, err)
		}
		return nil, errors.Wrapf(err, "read DelegateRewardSnapshot %s", delegateID)
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

// ErrNoPayoutRouting reports that the protocol holds no payout routing for
// this delegate at this height — it registered after the candidate index was
// refreshed, or is gone. Distinct from a failed read, which must not be
// mistaken for it.
var ErrNoPayoutRouting = errors.New("voter_reward: delegate has no payout routing")

// ReadPayoutAddress is the authoritative answer to "is this delegate opted
// in", and the only one that catches the Hermes migration at the fork block —
// that migration flips the opt-in bit inside CreatePreStates and emits no
// action and no receipt log, so an event-only indexer never sees it.
//
// Returns ErrNoPayoutRouting when the state simply does not exist; any other
// error is an infrastructure failure the caller must not swallow.
func ReadPayoutAddress(
	ctx context.Context, client iotexapi.APIServiceClient, delegateID string, height uint64,
) (*PayoutRouting, error) {
	ctx, cancel := context.WithTimeout(ctx, readStateTimeout)
	defer cancel()
	res, err := client.ReadState(ctx, &iotexapi.ReadStateRequest{
		ProtocolID: []byte(rewardingProtocolID),
		MethodName: []byte("DelegatePayoutAddress"),
		Arguments:  [][]byte{[]byte(delegateID)},
		Height:     fmt.Sprintf("%d", height),
	})
	if err != nil {
		if isStateNotFound(err) {
			return nil, errors.Wrapf(ErrNoPayoutRouting, "%s: %v", delegateID, err)
		}
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
