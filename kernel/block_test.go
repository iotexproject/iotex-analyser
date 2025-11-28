package kernel

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/v2/blockchain/filedao"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
	corecfg "github.com/iotexproject/iotex-core/v2/config"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
	"github.com/iotexproject/iotex-core/v2/testutil"
	"github.com/stretchr/testify/require"
)

func TestPutBlockAndGetBlockByHeight(t *testing.T) {
	require := require.New(t)
	testPath, err := testutil.PathOfTempFile("test-block")
	require.NoError(err)
	defer func() {
		testutil.CleanupPath(testPath)
	}()

	config.SetEVMNetworkID(4689)
	var tip protocol.TipInfo
	ctx := protocol.WithBlockchainCtx(
		genesis.WithGenesisContext(context.Background(), genesis.Default),
		protocol.BlockchainCtx{
			Tip: tip,
		},
	)
	cfg := corecfg.Default.DB
	cfg.DbPath = testPath
	cfg.ReadOnly = true
	deser := block.NewDeserializer(config.EVMNetworkID())
	fdao, err := filedao.NewFileDAO(cfg, deser)
	require.NoError(err)
	dao := blockdao.NewBlockDAOWithIndexersAndCache(fdao, nil, 100)
	require.NoError(dao.Start(ctx))
	defer func() {
		require.NoError(dao.Stop(ctx))
	}()

	config.Default.Iotex.ChainEndPoint = "api.iotex.one:80"
	blkCache, err := GetBlockByHeightFromChain(ctx, 11529321)
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
		blk1, err := GetBlockByHeightFromBlockDAO(uint64(i), NewLocalBatchBlockDao(dao))
		require.NoError(err)
		for k, r := range blk1.Receipts {
			require.Equal(blkCache.Receipts[k].ActionHash, r.ActionHash)
			require.Equal(blkCache.Receipts[k].TransactionLogs(), r.TransactionLogs())
		}
	}

	for i := 1; i <= numBlks; i++ {
		blk, err := GetBlockByHeightFromBlockDAO(uint64(i), NewLocalBatchBlockDao(dao))
		require.NoError(err)
		for k, r := range blk.Receipts {
			require.Equal(blkCache.Receipts[k].ActionHash, r.ActionHash)
			require.Equal(len(blkCache.Receipts[k].TransactionLogs()), len(r.TransactionLogs()))
		}
	}
}

func TestGetBlockByHeight(t *testing.T) {
	require := require.New(t)
	config.SetEVMNetworkID(4689)
	var tip protocol.TipInfo
	ctx := protocol.WithBlockchainCtx(
		genesis.WithGenesisContext(context.Background(), genesis.Default),
		protocol.BlockchainCtx{
			Tip: tip,
		},
	)
	cfg := corecfg.Default.DB
	cfg.DbPath = "/tmp/chain.db"
	deser := block.NewDeserializer(config.EVMNetworkID())
	dao, err := filedao.NewFileDAO(cfg, deser)
	require.NoError(dao.Start(ctx))
	defer func() {
		require.NoError(dao.Stop(ctx))
	}()
	blkHeight := uint64(11392458)
	blk, err := dao.GetBlockByHeight(blkHeight)
	require.NoError(err)
	require.Equal(len(blk.Receipts), 0)
	blk.Receipts, err = dao.GetReceipts(blkHeight)
	require.NoError(err)
	require.Equal(len(blk.Receipts), 3)
	for _, receipt := range blk.Receipts {
		require.Equal(len(receipt.TransactionLogs()), 0)
	}
	tlogs, err := dao.TransactionLogs(blkHeight)
	require.NoError(err)
	for _, l := range tlogs.Logs {
		if len(l.Transactions) == 0 {
			continue
		}
		l := l
		logs := make([]*action.TransactionLog, len(l.Transactions))
		for i, txn := range l.Transactions {
			i := i
			txn := txn
			amount, ok := new(big.Int).SetString(txn.Amount, 10)
			require.Equal(ok, true)
			logs[i] = &action.TransactionLog{
				Type:      txn.Type,
				Amount:    amount,
				Sender:    txn.Sender,
				Recipient: txn.Recipient,
			}
		}
		for k, j := range blk.Receipts {
			k := k
			j := j
			if j.ActionHash == hash.BytesToHash256(l.ActionHash) {
				if len(j.TransactionLogs()) == 0 {
					blk.Receipts[k] = j.AddTransactionLogs(logs...)
				}
			}
		}
	}
}
