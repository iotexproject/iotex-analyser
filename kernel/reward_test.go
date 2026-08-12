package kernel

import (
	"testing"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/rewardingpb"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// rewardLogsReceipt wraps the given reward logs into the RewardLogs payload
// shape that rewarding.UnmarshalRewardLog expects, and hangs it off a receipt.
func rewardLogsReceipt(t *testing.T, logs ...*rewardingpb.RewardLog) *action.Receipt {
	t.Helper()
	data, err := proto.Marshal(&rewardingpb.RewardLogs{Logs: logs})
	require.NoError(t, err)
	var actionHash hash.Hash256
	actionHash[0] = 0x7a
	r := &action.Receipt{Status: 1, ActionHash: actionHash}
	return r.AddLogs(&action.Log{Data: data})
}

// TestRewardInfoFromReceiptSkipsIIP59DiagnosticLogs pins the placement of the
// IIP-59 skip, not just its existence.
//
// EPOCH_DRAIN_OVERRUN and CURSOR_PROGRESS reuse RewardLog.addr for cursor
// bookkeeping rather than an address ("<era>:<remaining>" and
// "<era>:<delegate_idx>:<voter_idx>:<remaining>"). If the skip were written as a
// case in the accumulation switch instead of ahead of the map lookup, the map
// entry would already exist by then and every caller -- reward_history,
// block_reward -- iterates the map unconditionally, writing a row keyed by
// "12:0:0:41". Asserting the map has exactly one key is what catches that; a
// test that only asserted "no error" would pass with the bug present.
func TestRewardInfoFromReceiptSkipsIIP59DiagnosticLogs(t *testing.T) {
	r := require.New(t)
	const delegate = "io1jh0ekmccywfkmj7e8qsuzsupnlk3w5337hjjg2"

	receipt := rewardLogsReceipt(t,
		&rewardingpb.RewardLog{
			Type:   rewardingpb.RewardLog_EPOCH_REWARD,
			Addr:   delegate,
			Amount: "1000",
		},
		&rewardingpb.RewardLog{
			Type:   rewardingpb.RewardLog_CURSOR_PROGRESS,
			Addr:   "12:0:0:41",
			Amount: "0",
		},
		&rewardingpb.RewardLog{
			Type:   rewardingpb.RewardLog_EPOCH_DRAIN_OVERRUN,
			Addr:   "11:7",
			Amount: "123456",
		},
	)

	got, err := RewardInfoFromReceipt(receipt)
	r.NoError(err)
	r.Len(got, 1, "diagnostic logs must not materialize a map entry; got keys: %v", keysOf(got))
	r.Contains(got, delegate)
	r.Equal("1000", got[delegate].EpochReward.String())
	// The overrun residue must not leak into any payout bucket.
	r.Equal("0", got[delegate].BlockReward.String())
	r.Equal("0", got[delegate].FoundationBonus.String())
	r.Equal("0", got[delegate].PriorityBonus.String())
	r.Equal("0", got[delegate].UnproductiveSlash.String())
}

// TestRewardInfoFromReceiptUnknownTypeDoesNotHaltIndexer covers the default
// branch. A reward log type this build has never seen must be skipped, not
// turned into an error -- returning an error here halts the whole indexer at
// that block, which is the outage IIP-59's two new enum values would have
// caused against the previous code.
func TestRewardInfoFromReceiptUnknownTypeDoesNotHaltIndexer(t *testing.T) {
	r := require.New(t)
	const delegate = "io1jh0ekmccywfkmj7e8qsuzsupnlk3w5337hjjg2"

	receipt := rewardLogsReceipt(t,
		&rewardingpb.RewardLog{
			Type:   rewardingpb.RewardLog_BLOCK_REWARD,
			Addr:   delegate,
			Amount: "16",
		},
		&rewardingpb.RewardLog{
			// Not defined in any released core; stands in for the next enum
			// value a future release adds.
			Type:   rewardingpb.RewardLog_RewardType(99),
			Addr:   "whatever-this-carries",
			Amount: "not-a-number",
		},
	)

	got, err := RewardInfoFromReceipt(receipt)
	r.NoError(err)
	r.Len(got, 1, "unknown types must not materialize a map entry; got keys: %v", keysOf(got))
	r.Equal("16", got[delegate].BlockReward.String())
}

// TestRewardInfoFromReceiptAccumulatesKnownTypes guards the accumulation switch
// itself, so the classification switch added above cannot silently start
// dropping a real payout type.
func TestRewardInfoFromReceiptAccumulatesKnownTypes(t *testing.T) {
	r := require.New(t)
	const delegate = "io1jh0ekmccywfkmj7e8qsuzsupnlk3w5337hjjg2"

	receipt := rewardLogsReceipt(t,
		&rewardingpb.RewardLog{Type: rewardingpb.RewardLog_BLOCK_REWARD, Addr: delegate, Amount: "1"},
		&rewardingpb.RewardLog{Type: rewardingpb.RewardLog_EPOCH_REWARD, Addr: delegate, Amount: "2"},
		&rewardingpb.RewardLog{Type: rewardingpb.RewardLog_FOUNDATION_BONUS, Addr: delegate, Amount: "3"},
		&rewardingpb.RewardLog{Type: rewardingpb.RewardLog_PRIORITY_BONUS, Addr: delegate, Amount: "4"},
		&rewardingpb.RewardLog{Type: rewardingpb.RewardLog_UNPRODUCTIVE_SLASH, Addr: delegate, Amount: "5"},
	)

	got, err := RewardInfoFromReceipt(receipt)
	r.NoError(err)
	r.Len(got, 1)
	r.Equal("1", got[delegate].BlockReward.String())
	r.Equal("2", got[delegate].EpochReward.String())
	r.Equal("3", got[delegate].FoundationBonus.String())
	r.Equal("4", got[delegate].PriorityBonus.String())
	r.Equal("5", got[delegate].UnproductiveSlash.String())
}

// TestRewardInfoFromReceiptRejectsBadAmountOnPayoutType confirms the amount
// parse is still strict for types that do carry a payout -- the new skip logic
// must not have widened error tolerance for real rewards.
func TestRewardInfoFromReceiptRejectsBadAmountOnPayoutType(t *testing.T) {
	r := require.New(t)

	receipt := rewardLogsReceipt(t, &rewardingpb.RewardLog{
		Type:   rewardingpb.RewardLog_EPOCH_REWARD,
		Addr:   "io1jh0ekmccywfkmj7e8qsuzsupnlk3w5337hjjg2",
		Amount: "not-a-number",
	})

	_, err := RewardInfoFromReceipt(receipt)
	r.Error(err)
	r.Contains(err.Error(), "failed to convert reward amount")
}

func keysOf(m map[string]*RewardInfo) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
