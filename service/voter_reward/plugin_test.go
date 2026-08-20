package voter_reward

import (
	"testing"

	"github.com/iotexproject/iotex-analyser/plugin"
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

func TestHexOrEmpty(t *testing.T) {
	// Matches the protocol's own %x rendering of the cursor, so a value read
	// from state compares equal to one parsed out of a CURSOR_PROGRESS log.
	require.Equal(t, "", hexOrEmpty(nil))
	require.Equal(t, "", hexOrEmpty([]byte{}))
	require.Equal(t, "dead", hexOrEmpty([]byte{0xde, 0xad}))
}
