package voter_reward

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/distributedlog"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/rewardingpb"
	"github.com/iotexproject/iotex-core/v2/action/protocol/staking/freezelog"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func addr(t *testing.T, b byte) address.Address {
	t.Helper()
	raw := make([]byte, 20)
	for i := range raw {
		raw[i] = b
	}
	a, err := address.FromBytes(raw)
	require.NoError(t, err)
	return a
}

// rewardLog builds the shape the rewarding protocol writes: a marshalled
// RewardLog with no topics, emitted from the rewarding protocol address.
func rewardLog(t *testing.T, entries ...*rewardingpb.RewardLog) *action.Log {
	t.Helper()
	data, err := proto.Marshal(&rewardingpb.RewardLogs{Logs: entries})
	require.NoError(t, err)
	return &action.Log{Address: rewardingAddr, Data: data}
}

func TestDecodeDistributionFlattensParallelArrays(t *testing.T) {
	delegate := addr(t, 0xaa)
	voterA, voterB := addr(t, 0x01), addr(t, 0x02)
	recipientB := addr(t, 0x03)

	topics, data, err := distributedlog.Pack(distributedlog.EventArgs{
		Epoch:       7, // the ABI calls it epoch; the protocol fills it with the era
		Delegate:    delegate,
		VoterAmount: big.NewInt(300),
		Voters:      []address.Address{voterA, voterB},
		// A compounded payout still names the voter as recipient; a redirected
		// one names the destination.
		Recipients:        []address.Address{voterA, recipientB},
		Amounts:           []*big.Int{big.NewInt(100), big.NewInt(200)},
		CompoundBucketIDs: []uint64{0, 0},
		Compounded:        []bool{true, false},
	})
	require.NoError(t, err)

	rows, err := DecodeDistribution(&action.Log{Address: rewardingAddr, Topics: topics, Data: data})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	require.Equal(t, uint64(7), rows[0].Era)
	require.Equal(t, delegate.String(), rows[0].Delegate)
	require.Equal(t, voterA.String(), rows[0].Voter)
	require.Equal(t, "100", rows[0].Amount.String())
	require.Equal(t, uint32(0), rows[0].RowIndex)

	// Bucket 0 is a real native bucket. Both rows carry compoundBucketID 0 and
	// only the flag separates them — this is the assertion that guards the
	// single most likely misreading of this event.
	require.True(t, rows[0].Compounded)
	require.Equal(t, uint64(0), rows[0].CompoundBucketID)
	require.False(t, rows[1].Compounded)
	require.Equal(t, uint64(0), rows[1].CompoundBucketID)

	require.Equal(t, recipientB.String(), rows[1].Recipient)
	require.Equal(t, uint32(1), rows[1].RowIndex)
}

func TestDecodeDistributionIgnoresForeignLogs(t *testing.T) {
	delegate := addr(t, 0xaa)
	topics, data, err := distributedlog.Pack(distributedlog.EventArgs{
		Epoch: 1, Delegate: delegate, VoterAmount: big.NewInt(0),
	})
	require.NoError(t, err)

	// Right selector, wrong emitter: any contract may emit this topic, and
	// trusting it would let a user contract inject reward rows.
	rows, err := DecodeDistribution(&action.Log{
		Address: "io1qnpz47hx5q6r3w876axtrn6yz95d70cjl35r53", Topics: topics, Data: data,
	})
	require.NoError(t, err)
	require.Nil(t, rows)

	// A log with no topics at all is a RewardLog, not this event.
	rows, err = DecodeDistribution(&action.Log{Address: rewardingAddr, Data: data})
	require.NoError(t, err)
	require.Nil(t, rows)
}

func TestDecodeRewardBookkeepingCursorProgress(t *testing.T) {
	// Format comes from rewarding.encodeCursorProgressLog:
	// "%d:%d:%x:%d" over (targetEra, scanPhase, resumeVoter, rangesRemaining).
	l := rewardLog(t, &rewardingpb.RewardLog{
		Type:   rewardingpb.RewardLog_CURSOR_PROGRESS,
		Addr:   fmt.Sprintf("%d:%d:%x:%d", 12, ScanPhaseHead, []byte{0xde, 0xad}, 1),
		Amount: "0",
	})
	progress, overrun, err := DecodeRewardBookkeeping(l)
	require.NoError(t, err)
	require.Nil(t, overrun)
	require.NotNil(t, progress)
	require.Equal(t, uint64(12), progress.Era)
	require.Equal(t, ScanPhaseHead, progress.ScanPhase)
	require.Equal(t, "dead", progress.ResumeVoter)
	require.Equal(t, uint64(1), progress.RangesRemaining)
}

// TestDecodeRewardBookkeepingSignalsCompletion covers the settlement-complete
// path, which the local fixture never reaches: its per-block voter budget is
// deliberately tiny, so every era overruns instead of finishing.
func TestDecodeRewardBookkeepingSignalsCompletion(t *testing.T) {
	l := rewardLog(t, &rewardingpb.RewardLog{
		Type:   rewardingpb.RewardLog_CURSOR_PROGRESS,
		Addr:   fmt.Sprintf("%d:%d:%x:%d", 12, ScanPhaseDone, []byte{}, 0),
		Amount: "0",
	})
	progress, _, err := DecodeRewardBookkeeping(l)
	require.NoError(t, err)
	require.NotNil(t, progress)
	require.Equal(t, ScanPhaseDone, progress.ScanPhase)
	require.Equal(t, "", progress.ResumeVoter)
}

func TestDecodeRewardBookkeepingOverrun(t *testing.T) {
	l := rewardLog(t, &rewardingpb.RewardLog{
		Type:   rewardingpb.RewardLog_EPOCH_DRAIN_OVERRUN,
		Addr:   "12:3",
		Amount: "4200000000000000000",
	})
	progress, overrun, err := DecodeRewardBookkeeping(l)
	require.NoError(t, err)
	require.Nil(t, progress)
	require.NotNil(t, overrun)
	require.Equal(t, uint64(12), overrun.Era)
	require.Equal(t, uint64(3), overrun.DelegatesRemaining)
	require.Equal(t, "4200000000000000000", overrun.Residue.String())
}

// TestDecodeRewardBookkeepingIgnoresPayoutLogs is the regression guard for the
// bug that stopped the indexer at the fork: an ordinary BLOCK_REWARD entry
// must not be mistaken for one of the two bookkeeping types.
func TestDecodeRewardBookkeepingIgnoresPayoutLogs(t *testing.T) {
	l := rewardLog(t, &rewardingpb.RewardLog{
		Type:   rewardingpb.RewardLog_BLOCK_REWARD,
		Addr:   addr(t, 0x05).String(),
		Amount: "16000000000000000000",
	})
	progress, overrun, err := DecodeRewardBookkeeping(l)
	require.NoError(t, err)
	require.Nil(t, progress)
	require.Nil(t, overrun)
}

func TestDecodeRewardBookkeepingRejectsMalformedCursor(t *testing.T) {
	l := rewardLog(t, &rewardingpb.RewardLog{
		Type: rewardingpb.RewardLog_CURSOR_PROGRESS, Addr: "12:tail", Amount: "0",
	})
	_, _, err := DecodeRewardBookkeeping(l)
	require.Error(t, err)
	require.Contains(t, err.Error(), "want 4 fields")
}

func TestDecodeOptIn(t *testing.T) {
	candidate := addr(t, 0x77)
	// The protocol writes the identifier with hash.BytesToHash256, which
	// right-aligns it, even though the ABI declares the field as bytes32.
	topics := action.VoterRewardOptInSetEvent(candidate.Bytes())
	got, err := DecodeOptIn(&action.Log{Address: stakingAddr, Topics: topics})
	require.NoError(t, err)
	require.Equal(t, candidate.String(), got)

	// Same event from the wrong emitter is not ours.
	got, err = DecodeOptIn(&action.Log{Address: rewardingAddr, Topics: topics})
	require.NoError(t, err)
	require.Equal(t, "", got)
}

func TestDecodeDestination(t *testing.T) {
	voter, oldR, newR := addr(t, 0x11), addr(t, 0x22), addr(t, 0x33)
	topics, data, err := action.PackVoterRewardDestinationSetEvent(voter, oldR, newR)
	require.NoError(t, err)

	change, err := DecodeDestination(&action.Log{Address: rewardingAddr, Topics: topics, Data: data})
	require.NoError(t, err)
	require.NotNil(t, change)
	require.Equal(t, voter.String(), change.Voter)
	require.Equal(t, oldR.String(), change.OldRecipient)
	require.Equal(t, newR.String(), change.NewRecipient)
}

// freezeLog packs a DelegateRewardFrozen log the way poll_snapshot.go does.
func freezeLog(t *testing.T, args freezelog.EventArgs) *action.Log {
	t.Helper()
	topics, data, err := freezelog.Pack(args)
	require.NoError(t, err)
	return &action.Log{Address: stakingAddr, Topics: topics, Data: data}
}

func TestDecodeFreeze(t *testing.T) {
	r := require.New(t)
	delegate := addr(t, 0x42)
	// Every field a distinct value: the payload is four uint64s in a row, so
	// equal values would let a reordering pass unnoticed.
	got, err := DecodeFreeze(freezeLog(t, freezelog.EventArgs{
		Era:                  55080,
		Delegate:             delegate,
		FreezeHeight:         47169361,
		BlockCommissionBps:   3000,
		EpochCommissionBps:   3500,
		CommissionConfigured: true,
		TotalWeight:          big.NewInt(1234567890),
		SelfStakeBucketIdx:   91,
	}))
	r.NoError(err)
	r.NotNil(got)
	r.Equal(uint64(55080), got.Era)
	r.Equal(delegate.String(), got.Delegate)
	r.Equal(uint64(47169361), got.Snapshot.FreezeHeight)
	r.Equal(uint64(3000), got.Snapshot.BlockCommissionBps)
	r.Equal(uint64(3500), got.Snapshot.EpochCommissionBps)
	r.True(got.Snapshot.CommissionConfigured)
	r.Equal(0, got.Snapshot.TotalWeight.Cmp(big.NewInt(1234567890)))
	r.Equal(uint64(91), got.Snapshot.SelfStakeBucketIdx)
}

// A delegate that published no portions is frozen at 10000/10000, which by
// value is indistinguishable from one that deliberately takes everything.
// CommissionConfigured is the only thing that separates them, so it has to
// survive decoding as false rather than defaulting to true.
func TestDecodeFreezeCarriesUnconfiguredCommission(t *testing.T) {
	r := require.New(t)
	got, err := DecodeFreeze(freezeLog(t, freezelog.EventArgs{
		Era: 55080, Delegate: addr(t, 0x42), FreezeHeight: 47169361,
		BlockCommissionBps: 10000, EpochCommissionBps: 10000,
		CommissionConfigured: false, TotalWeight: big.NewInt(7),
	}))
	r.NoError(err)
	r.NotNil(got)
	r.False(got.Snapshot.CommissionConfigured)
	r.Equal(uint64(10000), got.Snapshot.BlockCommissionBps)
}

// A nil TotalWeight must not decode to a nil *big.Int: dec() would panic on it,
// and the protocol packs zero weight that way for a candidate with no votes.
func TestDecodeFreezeZeroWeightIsNotNil(t *testing.T) {
	r := require.New(t)
	got, err := DecodeFreeze(freezeLog(t, freezelog.EventArgs{
		Era: 1, Delegate: addr(t, 0x42), TotalWeight: nil,
	}))
	r.NoError(err)
	r.NotNil(got.Snapshot.TotalWeight)
	r.Equal(0, got.Snapshot.TotalWeight.Sign())
}

func TestDecodeFreezeIgnoresForeignLogs(t *testing.T) {
	r := require.New(t)
	good := freezeLog(t, freezelog.EventArgs{
		Era: 1, Delegate: addr(t, 0x42), TotalWeight: big.NewInt(1),
	})

	got, err := DecodeFreeze(nil)
	r.NoError(err)
	r.Nil(got)

	// Right topic, wrong emitter. Any contract may emit this selector.
	got, err = DecodeFreeze(&action.Log{
		Address: rewardingAddr, Topics: good.Topics, Data: good.Data})
	r.NoError(err)
	r.Nil(got, "only the staking protocol's own logs count")

	// The other staking events this plugin already reads must not match.
	optIn := action.VoterRewardOptInSetEvent(addr(t, 0x42).Bytes())
	got, err = DecodeFreeze(&action.Log{Address: stakingAddr, Topics: optIn})
	r.NoError(err)
	r.Nil(got)

	got, err = DecodeFreeze(&action.Log{
		Address: stakingAddr, Topics: good.Topics[:2], Data: good.Data})
	r.NoError(err)
	r.Nil(got, "era and delegate both live in topics; two is not enough")
}

// A truncated payload must be an error, not a zero-valued snapshot: the config
// row is written once per era and never revisited, so a silent zero would
// freeze a delegate at 0% commission forever.
func TestDecodeFreezeRejectsMalformedData(t *testing.T) {
	r := require.New(t)
	good := freezeLog(t, freezelog.EventArgs{
		Era: 1, Delegate: addr(t, 0x42), TotalWeight: big.NewInt(1),
	})
	_, err := DecodeFreeze(&action.Log{
		Address: stakingAddr, Topics: good.Topics, Data: []byte{0x01}})
	r.Error(err)
}
