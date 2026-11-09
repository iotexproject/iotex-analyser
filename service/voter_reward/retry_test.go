package voter_reward

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/distributedlog"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/rewardingpb"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
)

// fakeStateClient routes ReadState to a test-supplied function; every other
// API method panics through the nil embedded interface.
type fakeStateClient struct {
	iotexapi.APIServiceClient
	readState func(*iotexapi.ReadStateRequest) (*iotexapi.ReadStateResponse, error)
}

func (c *fakeStateClient) ReadState(
	_ context.Context, in *iotexapi.ReadStateRequest, _ ...grpc.CallOption,
) (*iotexapi.ReadStateResponse, error) {
	return c.readState(in)
}

func stateResponse(t *testing.T, m proto.Message) *iotexapi.ReadStateResponse {
	t.Helper()
	data, err := proto.Marshal(m)
	require.NoError(t, err)
	return &iotexapi.ReadStateResponse{Data: data}
}

// distributionClient answers VoterRewardDistribution for era. fail runs on
// every call first: returning an error fails the read, panicking panics it.
func distributionClient(
	t *testing.T, era uint64, fail func() error,
) *fakeStateClient {
	return &fakeStateClient{readState: func(in *iotexapi.ReadStateRequest) (*iotexapi.ReadStateResponse, error) {
		if string(in.GetMethodName()) != "VoterRewardDistribution" {
			return nil, errors.Errorf("unexpected ReadState %s", in.GetMethodName())
		}
		if err := fail(); err != nil {
			return nil, err
		}
		return stateResponse(t, &rewardingpb.VoterRewardDistributionState{
			TargetEra: era, ScanPhase: ScanPhaseTail,
		}), nil
	}}
}

// chunkHeights returns two consecutive heights at or after h that are neither
// an epoch's first nor last block, so putBlock's per-epoch passes stay out of
// the way and the only state read is the post-chunk cursor refresh.
func chunkHeights(h uint64) (uint64, uint64) {
	edge := func(x uint64) bool {
		e := kernel.GetEpochNum(x)
		return x == kernel.GetEpochHeight(e) || x == kernel.GetEpochLastBlockHeight(e)
	}
	for edge(h) || edge(h+1) {
		h += 3
	}
	return h, h + 1
}

// replayPlugin is a plugin whose era plan and candidate index were committed
// by an earlier batch, as they are once a settlement is a few chunks in.
func replayPlugin(client iotexapi.APIServiceClient, era, height uint64) Plugin {
	p := New(plugin.PluginSelf)
	p.chainClient = func() iotexapi.APIServiceClient { return client }
	p.eraSeen[era] = true
	p.candidates.height = height - 1
	return p
}

// requireSingleCount replays blks the way PutBlocks does, minus commit()
// (which needs a database; what it would write is exactly what is buffered),
// and checks that the era totals count every row exactly once.
func requireSingleCount(t *testing.T, p *Plugin, era uint64, blks []*block.Block, rows uint64, total int64) {
	t.Helper()
	for _, blk := range blks {
		require.NoError(t, p.putBlock(context.Background(), blk))
	}
	row := p.pendingEras[era]
	require.NotNil(t, row)
	require.Len(t, p.pendingRows, int(rows))
	require.Equal(t, rows, row.VoterCount)
	require.True(t, decimal.NewFromInt(total).Equal(row.TotalDistributed),
		"total_distributed = %s, want %d", row.TotalDistributed, total)
}

// chunkBlock builds a settlement continuation block as the protocol emits it:
// a CURSOR_PROGRESS reward log, then one DelegateVoterRewardsDistributed log.
func chunkBlock(t *testing.T, height, era uint64, amounts ...int64) *block.Block {
	t.Helper()
	var (
		voters  []address.Address
		amts    []*big.Int
		buckets []uint64
		comp    []bool
		total   int64
	)
	for i, a := range amounts {
		voters = append(voters, addr(t, byte(height%200)+byte(i)+1))
		amts = append(amts, big.NewInt(a))
		buckets = append(buckets, 0)
		comp = append(comp, false)
		total += a
	}
	topics, data, err := distributedlog.Pack(distributedlog.EventArgs{
		Epoch:             era,
		Delegate:          addr(t, 0xaa),
		VoterAmount:       big.NewInt(total),
		Voters:            voters,
		Recipients:        voters,
		Amounts:           amts,
		CompoundBucketIDs: buckets,
		Compounded:        comp,
	})
	require.NoError(t, err)

	receipt := (&action.Receipt{BlockHeight: height}).AddLogs(
		rewardLog(t, &rewardingpb.RewardLog{
			Type:   rewardingpb.RewardLog_CURSOR_PROGRESS,
			Addr:   fmt.Sprintf("%d:%d:%x:%d", era, ScanPhaseTail, []byte{0x01}, 1),
			Amount: "0",
		}),
		&action.Log{Address: rewardingAddr, Topics: topics, Data: data, BlockHeight: height},
	)
	blk, err := block.NewTestingBuilder().
		SetHeight(height).
		SetReceipts([]*action.Receipt{receipt}).
		SignAndBuild(identityset.PrivateKey(0))
	require.NoError(t, err)
	return &blk
}

// TestFailedBatchReplayDoesNotDoubleCountEraTotals is the regression guard for
// MainNet eras 65760, 65784 and 65808, whose voter_reward_era rows came out
// overstated by exactly the payout rows of the block a DeadlineExceeded hit.
//
// putBlock folds a block's rows into the era's running totals and only then
// re-reads the cursor. When that read fails, the runner replays the batch on
// the same instance; anything still buffered from the failed attempt gets
// counted again. The payout rows are deduplicated by their unique index on
// insert, the era totals are not.
func TestFailedBatchReplayDoesNotDoubleCountEraTotals(t *testing.T) {
	kernel.Init(&config.Default)

	const era = 65808
	// 54237248 is the MainNet block that was counted twice.
	h1, h2 := chunkHeights(54237248)
	blks := []*block.Block{
		chunkBlock(t, h1, era, 100, 200),
		chunkBlock(t, h2, era, 300, 400),
	}

	failures := 1
	client := distributionClient(t, era, func() error {
		if failures > 0 {
			failures--
			return status.Error(codes.DeadlineExceeded, "context deadline exceeded")
		}
		return nil
	})
	p := replayPlugin(client, era, h1)

	err := p.PutBlocks(context.Background(), blks)
	require.Error(t, err, "the first cursor read times out")
	require.Empty(t, p.pendingRows, "a failed batch must not leave rows for the replay to add to")
	require.Empty(t, p.pendingEras, "a failed batch must not leave era totals for the replay to add to")

	requireSingleCount(t, &p, era, blks, 4, 1000)
}

// The runner recovers a panic in PutBlocks and retries exactly as it does an
// error (server/runner.go putBlocks), so a panic must not leave the batch
// buffered either.
func TestPanickedBatchReplayDoesNotDoubleCountEraTotals(t *testing.T) {
	kernel.Init(&config.Default)

	const era = 65808
	h1, h2 := chunkHeights(54237248)
	blks := []*block.Block{
		chunkBlock(t, h1, era, 100, 200),
		chunkBlock(t, h2, era, 300, 400),
	}

	panics := 1
	client := distributionClient(t, era, func() error {
		if panics > 0 {
			panics--
			panic("node client blew up")
		}
		return nil
	})
	p := replayPlugin(client, era, h1)

	require.Panics(t, func() {
		//nolint:errcheck // the panic is the assertion
		p.PutBlocks(context.Background(), blks)
	})
	require.Empty(t, p.pendingRows, "a panicked batch must not leave rows for the replay to add to")
	require.Empty(t, p.pendingEras, "a panicked batch must not leave era totals for the replay to add to")

	requireSingleCount(t, &p, era, blks, 4, 1000)
}

// A delegate migrated at the fork block can still send SetVoterRewardOptIn
// later -- the protocol emits VoterRewardOptInSet even when the bit is already
// set. If a failed batch observed that action after an era's config pass, the
// replay's config pass must not credit the earlier era to it.
func TestOptInActionAfterTheSnapshotDoesNotRelabelTheEra(t *testing.T) {
	const (
		era    = 65808
		height = 54237240
	)
	delegate := addr(t, 0xaa)
	payout := addr(t, 0xbb)
	client := &fakeStateClient{readState: func(in *iotexapi.ReadStateRequest) (*iotexapi.ReadStateResponse, error) {
		if string(in.GetMethodName()) != "DelegatePayoutAddress" {
			return nil, errors.Errorf("unexpected ReadState %s", in.GetMethodName())
		}
		return stateResponse(t, &rewardingpb.DelegatePayoutAddress{
			Address: payout.Bytes(), OnchainRewardEnabled: true,
		}), nil
	}}

	for _, tc := range []struct {
		name       string
		seenAt     uint64
		wantSource string
		wantHeight uint64
	}{
		{"action after the snapshot", height + 5, optInSourceMigration, 0},
		{"action at the snapshot", height, optInSourceAction, height},
		{"action before the snapshot", height - 100, optInSourceAction, height - 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := New(plugin.PluginSelf)
			p.candidates.byID[delegate.String()] = CandidateInfo{Identifier: delegate.String(), Name: "d"}
			p.frozenSnaps[era] = map[string]*DelegateSnapshot{
				delegate.String(): {FreezeHeight: height - 2160, TotalWeight: big.NewInt(1)},
			}
			p.optInSeen[delegate.String()] = tc.seenAt

			_, optedIn, err := p.snapshotDelegateConfigs(context.Background(), client, era, height, nil)
			require.NoError(t, err)
			require.Equal(t, 1, optedIn)
			require.Len(t, p.pendingConfigs, 1)
			require.Equal(t, tc.wantSource, p.pendingConfigs[0].OptInSource)
			require.Equal(t, tc.wantHeight, p.pendingConfigs[0].OptInHeight)
		})
	}
}

func TestDiscardBatchClearsEverythingBuffered(t *testing.T) {
	p := New(plugin.PluginSelf)
	p.pendingRows = []models.VoterRewardDistribution{{}}
	p.pendingDestinations = []models.VoterRewardDestination{{}}
	p.pendingConfigs = []models.DelegateRewardConfig{{}}
	p.era(7).VoterCount = 3
	p.chunkEras[7] = struct{}{}
	p.pendingEraSeen[7] = true
	p.pendingEraDone[7] = true
	p.pendingEraStatePass[7] = true
	p.eraSeen[6] = true

	p.discardBatch()

	require.Empty(t, p.pendingRows)
	require.Empty(t, p.pendingDestinations)
	require.Empty(t, p.pendingConfigs)
	require.Empty(t, p.pendingEras)
	require.Empty(t, p.chunkEras)
	require.False(t, p.isEraSeen(7), "the replay has to redo the era plan it just discarded")
	require.False(t, p.isEraDone(7))
	require.False(t, p.isEraStatePassed(7))
	require.True(t, p.isEraSeen(6), "work committed by an earlier batch stays memoised")
}
