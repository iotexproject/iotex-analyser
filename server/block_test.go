package server

import (
	"context"
	"os"
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/action/protocol"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	corecfg "github.com/iotexproject/iotex-core/config"
	coreconfig "github.com/iotexproject/iotex-core/config"
	"github.com/stretchr/testify/require"
)

func TestTraceBlock(t *testing.T) {
	corecfg.SetEVMNetworkID(4689)
	//os.Setenv("TraceBlockConfig", "/Users/millken/work/iotex/iotex-analyser/config_mainnet.yml")
	_, err := config.New(os.Getenv("TraceBlockConfig"))
	require.NoError(t, err)

	var tip protocol.TipInfo
	ctxDao := protocol.WithBlockchainCtx(
		genesis.WithGenesisContext(context.Background(), genesis.Default),
		protocol.BlockchainCtx{
			Tip: tip,
		},
	)
	var indexers []blockdao.BlockIndexer
	deser := block.NewDeserializer(coreconfig.EVMNetworkID())
	dao := blockdao.NewBlockDAO(indexers, config.Default.BlockDB, deser)

	err = dao.Start(ctxDao)
	require.NoError(t, err)

	//defer dao.Stop(ctxDao)

	blk, err := dao.GetBlockByHeight(3)
	require.NoError(t, err)
	require.NotNil(t, blk)

	t.Logf("%s", getBlockString(blk))
}
