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

// flakyStateClient answers VoterRewardDistribution, failing the first
// `failures` calls the way a slow node does on MainNet.
type flakyStateClient struct {
	iotexapi.APIServiceClient
	failures int
	state    *rewardingpb.VoterRewardDistributionState
}

func (c *flakyStateClient) ReadState(
	_ context.Context, in *iotexapi.ReadStateRequest, _ ...grpc.CallOption,
) (*iotexapi.ReadStateResponse, error) {
	if string(in.GetMethodName()) != "VoterRewardDistribution" {
		return nil, errors.Errorf("unexpected ReadState %s", in.GetMethodName())
	}
	if c.failures > 0 {
		c.failures--
		return nil, status.Error(codes.DeadlineExceeded, "context deadline exceeded")
	}
	data, err := proto.Marshal(c.state)
	if err != nil {
		return nil, err
	}
	return &iotexapi.ReadStateResponse{Data: data}, nil
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
	height := uint64(54237248) // the MainNet block that was counted twice
	for height == kernel.GetEpochHeight(kernel.GetEpochNum(height)) ||
		height == kernel.GetEpochLastBlockHeight(kernel.GetEpochNum(height)) ||
		height+1 == kernel.GetEpochLastBlockHeight(kernel.GetEpochNum(height+1)) {
		height += 3
	}
	blks := []*block.Block{
		chunkBlock(t, height, era, 100, 200),
		chunkBlock(t, height+1, era, 300, 400),
	}

	client := &flakyStateClient{
		failures: 1,
		state:    &rewardingpb.VoterRewardDistributionState{TargetEra: era, ScanPhase: ScanPhaseTail},
	}
	p := New(plugin.PluginSelf)
	p.chainClient = func() iotexapi.APIServiceClient { return client }
	// The plan and the candidate index were committed by an earlier batch,
	// as they are by the time a settlement is a few chunks in.
	p.eraSeen[era] = true
	p.candidates.height = height - 1

	err := p.PutBlocks(context.Background(), blks)
	require.Error(t, err, "the first cursor read times out")
	require.Empty(t, p.pendingRows, "a failed batch must not leave rows for the replay to add to")
	require.Empty(t, p.pendingEras, "a failed batch must not leave era totals for the replay to add to")

	// The replay. putBlock is what PutBlocks runs per block; commit is left
	// out because it needs a database, and what it would write is exactly
	// what is buffered here.
	for _, blk := range blks {
		require.NoError(t, p.putBlock(context.Background(), blk))
	}
	row := p.pendingEras[era]
	require.NotNil(t, row)
	require.Len(t, p.pendingRows, 4)
	require.Equal(t, uint64(4), row.VoterCount)
	require.True(t, decimal.NewFromInt(1000).Equal(row.TotalDistributed),
		"total_distributed = %s, want 1000", row.TotalDistributed)
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
