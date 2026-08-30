// Package voter_reward decodes the IIP-59 on-chain voter reward distribution
// surface into indexable rows.
//
// Decoding goes through iotex-core's own exported helpers wherever they exist
// (distributedlog.Unpack/Topic0, action.VoterRewardOptInSetEvent,
// action.PackVoterRewardDestinationSetEvent). iotex-core exported the
// distributed-log API for exactly this reason: a vendored copy of the ABI
// silently decodes the wrong tuple the moment the protocol renames a field,
// because ABI decoding of a same-arity prefix does not fail.
//
// Two events — VoterRewardDestinationSet and DelegateRewardFrozen — have a Pack
// helper but no exported unpacker. The first is read positionally; the second
// is decoded against freezelog.ABIJSON, which iotex-core exports for exactly
// this purpose. selfCheck() below round-trips the protocol's own Pack through
// both readers at init, so a protocol-side layout change fails loudly at
// startup instead of quietly producing wrong values.
package voter_reward

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/distributedlog"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/rewardingpb"
	"github.com/iotexproject/iotex-core/v2/action/protocol/staking"
	"github.com/iotexproject/iotex-core/v2/action/protocol/staking/freezelog"
	"github.com/pkg/errors"
)

const (
	// ScanPhaseTail scans [startVoter, max], ScanPhaseHead wraps to
	// [min, startVoter), ScanPhaseDone means the settlement is complete.
	ScanPhaseTail uint32 = 0
	ScanPhaseHead uint32 = 1
	ScanPhaseDone uint32 = 2

	// addrOffset is where a 20-byte address sits inside a 32-byte topic.
	// Both indexed values this package reads are written with
	// hash.BytesToHash256, which right-aligns — including
	// VoterRewardOptInSet's candidateIdentifier, which the ABI declares as
	// bytes32 but the protocol fills with a right-aligned address.
	addrOffset = 12
)

var (
	rewardingAddr = rewarding.ProtocolAddr().String()
	stakingAddr   = staking.ProtocolAddr().String()

	distributedTopic0 hash.Hash256
	optInTopic0       hash.Hash256
	destinationTopic0 hash.Hash256

	// freezeEvent is the protocol's own ABI declaration of
	// DelegateRewardFrozen, parsed from the constant iotex-core exports for
	// off-chain consumers rather than from a signature string copied here.
	freezeEvent  abi.Event
	freezeTopic0 hash.Hash256
)

func init() {
	var err error
	distributedTopic0, err = distributedlog.Topic0()
	if err != nil {
		panic("voter_reward: load DelegateVoterRewardsDistributed topic: " + err.Error())
	}
	// Derive both remaining selectors from the protocol's own packers rather
	// than re-deriving them from a copied signature string.
	optInTopic0 = action.VoterRewardOptInSetEvent(make([]byte, 20))[0]
	probe, err := address.FromBytes(make([]byte, 20))
	if err != nil {
		panic("voter_reward: build probe address: " + err.Error())
	}
	topics, _, err := action.PackVoterRewardDestinationSetEvent(probe, probe, probe)
	if err != nil {
		panic("voter_reward: derive VoterRewardDestinationSet topic: " + err.Error())
	}
	destinationTopic0 = topics[0]
	freezeABI, err := abi.JSON(strings.NewReader(freezelog.ABIJSON))
	if err != nil {
		panic("voter_reward: parse freezelog ABI: " + err.Error())
	}
	var ok bool
	freezeEvent, ok = freezeABI.Events[freezelog.EventName]
	if !ok {
		panic("voter_reward: event " + freezelog.EventName + " absent from freezelog ABI")
	}
	freezeTopic0 = hash.Hash256(freezeEvent.ID)
	if err := selfCheck(); err != nil {
		panic("voter_reward: " + err.Error())
	}
}

// selfCheck asserts that the positional reader in DecodeDestination still
// agrees with the protocol's packer.
func selfCheck() error {
	voter, err := address.FromBytes(bytes.Repeat([]byte{0x11}, 20))
	if err != nil {
		return err
	}
	oldR, err := address.FromBytes(bytes.Repeat([]byte{0x22}, 20))
	if err != nil {
		return err
	}
	newR, err := address.FromBytes(bytes.Repeat([]byte{0x33}, 20))
	if err != nil {
		return err
	}
	topics, data, err := action.PackVoterRewardDestinationSetEvent(voter, oldR, newR)
	if err != nil {
		return err
	}
	got, err := DecodeDestination(&action.Log{Address: rewardingAddr, Topics: topics, Data: data})
	if err != nil {
		return errors.Wrap(err, "VoterRewardDestinationSet self-check")
	}
	if got == nil || got.Voter != voter.String() ||
		got.OldRecipient != oldR.String() || got.NewRecipient != newR.String() {
		return errors.Errorf(
			"VoterRewardDestinationSet layout changed in iotex-core; decoded %+v", got)
	}
	return freezeSelfCheck(voter)
}

// freezeSelfCheck round-trips freezelog.Pack through DecodeFreeze.
//
// The values are deliberately all distinct: DelegateRewardFrozen's payload is
// four uint64s in a row, so a reordering upstream would decode cleanly and
// silently swap, say, the freeze height with a commission rate. Distinct
// values turn that into a startup failure.
func freezeSelfCheck(delegate address.Address) error {
	want := freezelog.EventArgs{
		Era:                  7,
		Delegate:             delegate,
		FreezeHeight:         11,
		BlockCommissionBps:   13,
		EpochCommissionBps:   17,
		CommissionConfigured: true,
		TotalWeight:          big.NewInt(19),
		SelfStakeBucketIdx:   23,
	}
	topics, data, err := freezelog.Pack(want)
	if err != nil {
		return errors.Wrap(err, "pack DelegateRewardFrozen")
	}
	got, err := DecodeFreeze(&action.Log{Address: stakingAddr, Topics: topics, Data: data})
	if err != nil {
		return errors.Wrap(err, "DelegateRewardFrozen self-check")
	}
	if got == nil || got.Era != want.Era || got.Delegate != delegate.String() ||
		got.Snapshot.FreezeHeight != want.FreezeHeight ||
		got.Snapshot.BlockCommissionBps != want.BlockCommissionBps ||
		got.Snapshot.EpochCommissionBps != want.EpochCommissionBps ||
		!got.Snapshot.CommissionConfigured ||
		got.Snapshot.TotalWeight.Cmp(want.TotalWeight) != 0 ||
		got.Snapshot.SelfStakeBucketIdx != want.SelfStakeBucketIdx {
		return errors.Errorf(
			"DelegateRewardFrozen layout changed in iotex-core; decoded %+v", got)
	}
	return nil
}

// RewardingAddr is the protocol address every IIP-59 reward event is emitted
// from. A log carrying a matching selector but a different emitter is ignored:
// any user contract is free to emit the same topic.
func RewardingAddr() string { return rewardingAddr }

// StakingAddr is the emitter of VoterRewardOptInSet.
func StakingAddr() string { return stakingAddr }

// DistributionRow is one voter's share inside one
// DelegateVoterRewardsDistributed log.
type DistributionRow struct {
	Era              uint64
	Delegate         string
	Voter            string
	Recipient        string
	Amount           *big.Int
	Compounded       bool
	CompoundBucketID uint64
	// RowIndex is the position inside the log's parallel arrays. With the log
	// index it makes a row addressable, which is what re-indexing a block
	// idempotently needs.
	RowIndex uint32
}

// DecodeDistribution expands one DelegateVoterRewardsDistributed log into one
// row per voter. It returns (nil, nil) for a log this package does not own, so
// a caller can pass every receipt log through it.
func DecodeDistribution(l *action.Log) ([]DistributionRow, error) {
	if l == nil || len(l.Topics) == 0 || l.Topics[0] != distributedTopic0 || l.Address != rewardingAddr {
		return nil, nil
	}
	args, err := distributedlog.Unpack(l.Topics, l.Data)
	if err != nil {
		return nil, errors.Wrap(err, "decode DelegateVoterRewardsDistributed")
	}
	rows := make([]DistributionRow, 0, len(args.Voters))
	for i := range args.Voters {
		rows = append(rows, DistributionRow{
			// The ABI names the first indexed field `epoch`, but
			// packDelegateChunkLog fills it with the target era. Recording it
			// as the era is what joins the row to its settlement.
			Era:      args.Epoch,
			Delegate: args.Delegate.String(),
			Voter:    args.Voters[i].String(),
			// For a compounded payout the protocol still fills recipients[i]
			// with the voter, so this is never empty.
			Recipient:        args.Recipients[i].String(),
			Amount:           args.Amounts[i],
			Compounded:       args.Compounded[i],
			CompoundBucketID: args.CompoundBucketIDs[i],
			RowIndex:         uint32(i),
		})
	}
	return rows, nil
}

// DecodeOptIn returns the candidate identifier from a VoterRewardOptInSet log,
// or "" for any other log.
func DecodeOptIn(l *action.Log) (string, error) {
	if l == nil || len(l.Topics) < 2 || l.Topics[0] != optInTopic0 || l.Address != stakingAddr {
		return "", nil
	}
	addr, err := address.FromBytes(l.Topics[1][addrOffset:])
	if err != nil {
		return "", errors.Wrap(err, "decode VoterRewardOptInSet candidate identifier")
	}
	return addr.String(), nil
}

// FrozenSnapshot is one DelegateRewardFrozen log: the reward policy the
// protocol locked in for one delegate for one era.
//
// It carries exactly what ReadDelegateSnapshot returns from chain state, which
// is the point — the event is the in-block source for the same values, so the
// state read stays available as a fallback for chains where the log is not
// emitted.
type FrozenSnapshot struct {
	Era      uint64
	Delegate string
	Snapshot *DelegateSnapshot
}

// DecodeFreeze returns the frozen policy carried by a DelegateRewardFrozen log,
// or nil for any other log.
//
// The protocol emits one of these per opted-in delegate, all within the single
// block at the era's freeze height, and only once EmitEraFreezeLog is active.
// Before that fork gate — and on any era whose freeze block this process did
// not observe — there are no logs to read and the caller falls back to state.
func DecodeFreeze(l *action.Log) (*FrozenSnapshot, error) {
	if l == nil || len(l.Topics) < 3 || l.Topics[0] != freezeTopic0 || l.Address != stakingAddr {
		return nil, nil
	}
	// Topics[1] is a uint64 left-padded to 32 bytes, so the value is in the
	// last 8; Topics[2] is an address right-aligned by hash.BytesToHash256.
	era := binary.BigEndian.Uint64(l.Topics[1][24:])
	delegate, err := address.FromBytes(l.Topics[2][addrOffset:])
	if err != nil {
		return nil, errors.Wrap(err, "decode DelegateRewardFrozen delegate")
	}
	args, err := freezeEvent.Inputs.NonIndexed().Unpack(l.Data)
	if err != nil {
		return nil, errors.Wrap(err, "decode DelegateRewardFrozen data")
	}
	if len(args) != 6 {
		return nil, errors.Errorf("DelegateRewardFrozen: want 6 data fields, got %d", len(args))
	}
	snap := &DelegateSnapshot{}
	// Checked one by one rather than with a bare type assertion: the six fields
	// are uint64/uint64/uint64/bool/uint256/uint64, and a same-arity reordering
	// upstream would otherwise panic in production instead of erroring here.
	// freezeSelfCheck catches such a change at startup; this is the backstop.
	var ok bool
	if snap.FreezeHeight, ok = args[0].(uint64); !ok {
		return nil, errors.Errorf("DelegateRewardFrozen: freezeHeight is %T", args[0])
	}
	if snap.BlockCommissionBps, ok = args[1].(uint64); !ok {
		return nil, errors.Errorf("DelegateRewardFrozen: blockCommissionBps is %T", args[1])
	}
	if snap.EpochCommissionBps, ok = args[2].(uint64); !ok {
		return nil, errors.Errorf("DelegateRewardFrozen: epochCommissionBps is %T", args[2])
	}
	if snap.CommissionConfigured, ok = args[3].(bool); !ok {
		return nil, errors.Errorf("DelegateRewardFrozen: commissionConfigured is %T", args[3])
	}
	if snap.TotalWeight, ok = args[4].(*big.Int); !ok {
		return nil, errors.Errorf("DelegateRewardFrozen: totalWeight is %T", args[4])
	}
	if snap.SelfStakeBucketIdx, ok = args[5].(uint64); !ok {
		return nil, errors.Errorf("DelegateRewardFrozen: selfStakeBucketIdx is %T", args[5])
	}
	return &FrozenSnapshot{Era: era, Delegate: delegate.String(), Snapshot: snap}, nil
}

// DestinationChange is one VoterRewardDestinationSet event.
type DestinationChange struct {
	Voter        string
	OldRecipient string
	NewRecipient string
}

// DecodeDestination returns nil for any log that is not a
// VoterRewardDestinationSet emitted by the rewarding protocol.
func DecodeDestination(l *action.Log) (*DestinationChange, error) {
	if l == nil || len(l.Topics) < 2 || l.Topics[0] != destinationTopic0 || l.Address != rewardingAddr {
		return nil, nil
	}
	voter, err := address.FromBytes(l.Topics[1][addrOffset:])
	if err != nil {
		return nil, errors.Wrap(err, "decode VoterRewardDestinationSet voter")
	}
	// Two non-indexed addresses, each left-padded to 32 bytes. There is no
	// exported unpacker for this event; selfCheck() guards the assumption.
	if len(l.Data) != 64 {
		return nil, errors.Errorf("VoterRewardDestinationSet: want 64 data bytes, got %d", len(l.Data))
	}
	oldR, err := address.FromBytes(l.Data[addrOffset:32])
	if err != nil {
		return nil, errors.Wrap(err, "decode VoterRewardDestinationSet oldRecipient")
	}
	newR, err := address.FromBytes(l.Data[32+addrOffset:])
	if err != nil {
		return nil, errors.Wrap(err, "decode VoterRewardDestinationSet newRecipient")
	}
	return &DestinationChange{
		Voter:        voter.String(),
		OldRecipient: oldR.String(),
		NewRecipient: newR.String(),
	}, nil
}

// CursorProgress is the settlement checkpoint every chunk prepends to its
// receipt as a CURSOR_PROGRESS reward log. Reading it is what lets the indexer
// follow a drain without polling contract state on every block.
type CursorProgress struct {
	Era             uint64
	ScanPhase       uint32
	ResumeVoter     string
	RangesRemaining uint64
}

// DrainOverrun is an EPOCH_DRAIN_OVERRUN reward log: a settlement was still
// live when the next era boundary arrived. The residue is not lost — it rolls
// into the next settlement — but a non-zero value means the drain fell behind.
type DrainOverrun struct {
	Era                uint64
	DelegatesRemaining uint64
	Residue            *big.Int
}

// DecodeRewardBookkeeping pulls the two non-monetary IIP-59 log types out of a
// GrantReward receipt log. Both reuse the RewardLog wire shape and pack
// structured text into its Addr field, so they are parsed positionally against
// the format strings in rewarding.encodeCursorProgressLog /
// encodeOverrunLog.
//
// A log that is not a reward log at all yields (nil, nil, nil): callers pass
// every receipt log through here.
func DecodeRewardBookkeeping(l *action.Log) (*CursorProgress, *DrainOverrun, error) {
	if l == nil || l.Address != rewardingAddr || len(l.Topics) != 0 {
		return nil, nil, nil
	}
	logs, err := rewarding.UnmarshalRewardLog(l.Data)
	if err != nil {
		return nil, nil, nil
	}
	var (
		progress *CursorProgress
		overrun  *DrainOverrun
	)
	for _, rl := range logs.Logs {
		switch rl.Type {
		case rewardingpb.RewardLog_CURSOR_PROGRESS:
			// "<targetEra>:<scanPhase>:<resumeVoterHex>:<rangesRemaining>".
			// resumeVoterHex is %x of the raw cursor bytes and is empty at the
			// start of a phase, so SplitN keeps a trailing empty field.
			parts := strings.Split(rl.Addr, ":")
			if len(parts) != 4 {
				return nil, nil, errors.Errorf("CURSOR_PROGRESS: want 4 fields, got %q", rl.Addr)
			}
			era, err := strconv.ParseUint(parts[0], 10, 64)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "CURSOR_PROGRESS era %q", parts[0])
			}
			phase, err := strconv.ParseUint(parts[1], 10, 32)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "CURSOR_PROGRESS scanPhase %q", parts[1])
			}
			ranges, err := strconv.ParseUint(parts[3], 10, 64)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "CURSOR_PROGRESS rangesRemaining %q", parts[3])
			}
			progress = &CursorProgress{
				Era: era, ScanPhase: uint32(phase),
				ResumeVoter: parts[2], RangesRemaining: ranges,
			}
		case rewardingpb.RewardLog_EPOCH_DRAIN_OVERRUN:
			// "<targetEra>:<delegatesRemaining>", amount = residue.
			parts := strings.Split(rl.Addr, ":")
			if len(parts) != 2 {
				return nil, nil, errors.Errorf("EPOCH_DRAIN_OVERRUN: want 2 fields, got %q", rl.Addr)
			}
			era, err := strconv.ParseUint(parts[0], 10, 64)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "EPOCH_DRAIN_OVERRUN era %q", parts[0])
			}
			remaining, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "EPOCH_DRAIN_OVERRUN delegatesRemaining %q", parts[1])
			}
			residue, ok := new(big.Int).SetString(rl.Amount, 10)
			if !ok {
				return nil, nil, errors.Errorf("EPOCH_DRAIN_OVERRUN residue %q", rl.Amount)
			}
			overrun = &DrainOverrun{Era: era, DelegatesRemaining: remaining, Residue: residue}
		}
	}
	return progress, overrun, nil
}
