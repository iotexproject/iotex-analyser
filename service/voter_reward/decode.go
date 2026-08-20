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
// One event — VoterRewardDestinationSet — has a Pack helper but no exported
// unpacker, so its two-address payload is read positionally. selfCheck() below
// round-trips the protocol's own Pack through that reader at init, so a
// protocol-side layout change fails loudly at startup instead of quietly
// producing wrong recipients.
package voter_reward

import (
	"bytes"
	"math/big"
	"strconv"
	"strings"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/distributedlog"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/rewardingpb"
	"github.com/iotexproject/iotex-core/v2/action/protocol/staking"
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
