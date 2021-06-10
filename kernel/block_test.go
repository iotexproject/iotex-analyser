package kernel

import (
	"context"
	"testing"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/action/protocol"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	corecfg "github.com/iotexproject/iotex-core/config"
	"github.com/iotexproject/iotex-core/test/identityset"
	"github.com/iotexproject/iotex-core/testutil"
	"github.com/stretchr/testify/require"
)

func TestPutBlockAndGetBlockByHeight(t *testing.T) {
	require := require.New(t)
	testPath, err := testutil.PathOfTempFile("test-block")
	require.NoError(err)
	defer func() {
		testutil.CleanupPath(t, testPath)
	}()

	corecfg.SetEVMNetworkID(4689)
	var tip protocol.TipInfo
	ctx := protocol.WithBlockchainCtx(
		context.Background(),
		protocol.BlockchainCtx{
			Genesis: genesis.Default,
			Tip:     tip,
		},
	)
	var indexers []blockdao.BlockIndexer
	var dao blockdao.BlockDAO
	cfg := corecfg.Default.DB
	cfg.DbPath = testPath
	dao = blockdao.NewBlockDAO(indexers, cfg)
	require.NoError(dao.Start(ctx))
	defer func() {
		require.NoError(dao.Stop(ctx))
	}()

	config.Default.Iotex.ChainEndPoint = "api.mainnet.iotex.one:80"
	blkCache, err := GetBlockByHeightFromChain(11529321)
	require.NoError(err)

	prevHash := hash.ZeroHash256
	numBlks := 32
	for i := 1; i <= numBlks; i++ {
		tb := block.TestingBuilder{}
		blk, err := tb.SetPrevBlockHash(prevHash).
			SetVersion(1).
			SetTimeStamp(time.Now()).
			SetHeight(uint64(i)).
			SetReceipts(blkCache.Receipts).
			SignAndBuild(identityset.PrivateKey(0))
		require.NoError(err)
		err = dao.PutBlock(ctx, &blk)
		require.NoError(err)
		prevHash = blk.HashBlock()
		blk1, err := GetBlockByHeightFromBlockDAO(uint64(i), dao)
		require.NoError(err)
		for k, r := range blk1.Receipts {
			require.Equal(blkCache.Receipts[k].ActionHash, r.ActionHash)
			require.Equal(blkCache.Receipts[k].TransactionLogs(), r.TransactionLogs())
		}
	}

	for i := 1; i <= numBlks; i++ {
		blk, err := GetBlockByHeightFromBlockDAO(uint64(i), dao)
		require.NoError(err)
		for k, r := range blk.Receipts {
			require.Equal(blkCache.Receipts[k].ActionHash, r.ActionHash)
			require.Equal(len(blkCache.Receipts[k].TransactionLogs()), len(r.TransactionLogs()))
		}
	}
}
