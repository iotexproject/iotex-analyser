package main

import (
	"context"
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/stretchr/testify/require"
)

func TestERC1155(t *testing.T) {
	var err error
	require := require.New(t)
	//os.Setenv("ConfigPath", "/Users/millken/work/iotex/iotex-analyser/config_mainlive.yml")
	config.SetEVMNetworkID(4689)
	_, err = db.LoadDBFromEnv()
	require.NoError(err)
	ctx := context.Background()
	blk, err := kernel.GetBlockByHeightFromChain(ctx, 21544145)
	require.NoError(err)
	require.NoError(Plugin.Start(ctx))
	err = Plugin.PutBlock(ctx, blk)
	require.NoError(err)
	require.NoError(Plugin.Stop(ctx))
}
