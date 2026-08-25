package voter_reward

import (
	"testing"

	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// The era memos gate expensive, once-per-era work: ensureEraPlan reads the
// settlement plan and sweeps every candidate's routing, and refreshChunkedEras
// stops re-reading state once a settlement is complete. Both are skipped when
// the memo says the work is done — so a memo that outlives a failed commit
// makes the runner's replay skip an era plan that was never written.

func TestPendingMemosGateWorkButOnlyCommitMakesThemDurable(t *testing.T) {
	p := New(plugin.PluginSelf)

	// Work done while building a batch is memoised immediately, so the rest of
	// the batch does not redo it.
	p.pendingEraSeen[7] = true
	p.pendingEraDone[7] = true
	require.True(t, p.isEraSeen(7))
	require.True(t, p.isEraDone(7))

	// ...but it is not durable until the transaction lands.
	require.False(t, p.eraSeen[7])
	require.False(t, p.eraDone[7])

	p.promotePendingMemos()
	require.True(t, p.isEraSeen(7))
	require.True(t, p.isEraDone(7))
	require.True(t, p.eraSeen[7], "a committed batch's memo must survive into the durable map")
	require.Empty(t, p.pendingEraSeen)
	require.Empty(t, p.pendingEraDone)
}

// TestDroppedMemosLetTheReplayRedoTheWork is the regression guard for the
// failure that leaves an era permanently at freeze_height=0: PutBlocks returns
// an error, the runner replays the same batch without advancing nextHeight,
// and ensureEraPlan must not treat the era as already summarised.
func TestDroppedMemosLetTheReplayRedoTheWork(t *testing.T) {
	p := New(plugin.PluginSelf)

	p.pendingEraSeen[7] = true
	p.pendingEraDone[7] = true

	p.dropPendingMemos()

	require.False(t, p.isEraSeen(7), "a failed commit must not leave the era looking summarised")
	require.False(t, p.isEraDone(7), "a failed commit must not leave the settlement looking complete")
	require.Empty(t, p.pendingEraSeen)
	require.Empty(t, p.pendingEraDone)
}

// Memos promoted by an earlier successful batch stay put when a later batch
// fails — only the failing batch's own work is redone.
func TestDropPendingMemosKeepsAlreadyCommittedEras(t *testing.T) {
	p := New(plugin.PluginSelf)

	p.pendingEraSeen[6] = true
	p.promotePendingMemos()

	p.pendingEraSeen[7] = true
	p.dropPendingMemos()

	require.True(t, p.isEraSeen(6))
	require.False(t, p.isEraSeen(7))
}

// TestStatePassMemoDoesNotSuppressTheCursorPath guards the regression that left
// three settled eras on TestNet with first_chunk_at=0, total_frozen=0, a
// delegate_count that meant something else, and every delegate_reward_config
// row at voter_amount_frozen=0 despite 185k IOTX having been paid out.
//
// indexFrozenEra runs on the era-boundary block and the first chunk lands on
// the very next one. While both shared the eraSeen memo, the boundary pass
// always won the race and ensureEraPlan skipped every era that had a cursor —
// so the half of the row only the cursor can supply was never written.
func TestStatePassMemoDoesNotSuppressTheCursorPath(t *testing.T) {
	p := New(plugin.PluginSelf)

	// The boundary pass ran and claimed its own memo.
	p.pendingEraStatePass[7] = true

	require.True(t, p.isEraStatePassed(7), "the boundary pass must not repeat itself")
	require.False(t, p.isEraSeen(7),
		"the boundary pass must leave ensureEraPlan free to run when a chunk arrives")

	// And the reverse: once the cursor path has recorded the plan, the boundary
	// pass has nothing to add and stands down.
	q := New(plugin.PluginSelf)
	q.pendingEraSeen[7] = true
	require.False(t, q.isEraStatePassed(7))
	require.True(t, q.isEraSeen(7))
}

// The new memo follows the same commit discipline as the other two: a batch
// that fails to land must not leave the era looking already swept.
func TestStatePassMemoIsOnlyDurableAfterCommit(t *testing.T) {
	p := New(plugin.PluginSelf)

	p.pendingEraStatePass[7] = true
	require.True(t, p.isEraStatePassed(7))
	require.False(t, p.eraStatePass[7])

	p.dropPendingMemos()
	require.False(t, p.isEraStatePassed(7), "a failed commit must not leave the era looking swept")

	p.pendingEraStatePass[8] = true
	p.promotePendingMemos()
	require.True(t, p.eraStatePass[8])
	require.Empty(t, p.pendingEraStatePass)
}

func TestHexOrEmpty(t *testing.T) {
	// Matches the protocol's own %x rendering of the cursor, so a value read
	// from state compares equal to one parsed out of a CURSOR_PROGRESS log.
	require.Equal(t, "", hexOrEmpty(nil))
	require.Equal(t, "", hexOrEmpty([]byte{}))
	require.Equal(t, "dead", hexOrEmpty([]byte{0xde, 0xad}))
}

// TestDedupeConfigsKeepsTheLastPerDelegateEra is the regression guard for a
// replay that could never finish.
//
// Both passes that write an era's configs can land in the same batch: the
// boundary pass runs on the block closing the era, the first chunk on the very
// next one, and a 200-block batch routinely spans both. Postgres rejects an
// INSERT .. ON CONFLICT DO UPDATE naming the same conflict target twice, the
// runner replays that batch, and the plugin wedges -- observed on a test network
// stuck at height 46,929,480 retrying forever.
func TestDedupeConfigsKeepsTheLastPerDelegateEra(t *testing.T) {
	r := require.New(t)

	// The boundary pass has no frozen amount; the cursor path that follows it
	// does, and that is the row that has to survive.
	fromBoundary := models.DelegateRewardConfig{DelegateID: "io1a", Era: 54912, BlockHeight: 46929600}
	fromCursor := models.DelegateRewardConfig{DelegateID: "io1a", Era: 54912, BlockHeight: 46929601,
		VoterAmountFrozen: decimal.NewFromInt(300)}
	otherDelegate := models.DelegateRewardConfig{DelegateID: "io1b", Era: 54912, BlockHeight: 46929600}
	otherEra := models.DelegateRewardConfig{DelegateID: "io1a", Era: 54936, BlockHeight: 46964160}

	out := dedupeConfigs([]models.DelegateRewardConfig{fromBoundary, otherDelegate, fromCursor, otherEra})

	r.Len(out, 3, "one row per (delegate, era)")
	r.Equal(uint64(46929601), out[0].BlockHeight, "the later write wins")
	r.True(out[0].VoterAmountFrozen.Equal(decimal.NewFromInt(300)),
		"the cursor path's frozen amount is the whole reason it wins")
	// Order is preserved, so the collapsed row stays where it first appeared.
	r.Equal("io1b", out[1].DelegateID)
	r.Equal(uint64(54936), out[2].Era)
}

func TestDedupeConfigsPassesThroughShortInput(t *testing.T) {
	r := require.New(t)
	r.Empty(dedupeConfigs(nil))
	one := []models.DelegateRewardConfig{{DelegateID: "io1a", Era: 1}}
	r.Len(dedupeConfigs(one), 1)
}
