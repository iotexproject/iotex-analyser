package server

import (
	"context"
	"os"
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/v2/action/protocol"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/blockchain/filedao"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
	"github.com/stretchr/testify/require"
)

func TestTraceBlock(t *testing.T) {
	config.SetEVMNetworkID(4689)
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
	deser := block.NewDeserializer(config.EVMNetworkID())
	dao, err := filedao.NewFileDAO(config.Default.BlockDB.Config, deser)
	require.NoError(t, err)
	err = dao.Start(ctxDao)
	require.NoError(t, err)

	//defer dao.Stop(ctxDao)

	blk, err := dao.GetBlockByHeight(3)
	require.NoError(t, err)
	require.NotNil(t, blk)

	t.Logf("%s", getBlockString(blk))
}
