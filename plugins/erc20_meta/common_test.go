package main

import (
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/stretchr/testify/require"
)

func TestReadErc20(t *testing.T) {
	require := require.New(t)
	config.Default.Iotex.ChainInsecure = false
	config.Default.Iotex.ChainEndPoint = "api.mainnet.iotex.one:443"
	client := kernel.ChainClient()
	require.NoError(initErc20())
	t.Run("ReadERC20Decimals", func(t *testing.T) {
		decimals, err := ReadERC20Decimals(client, "io15uggvd649nk8arpd6z9fekv4etlcks5qwdy3pz")
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(18, decimals)
	})
	t.Run("ReadERC20Symbol", func(t *testing.T) {
		symbol, err := ReadERC20Symbol(client, "io15uggvd649nk8arpd6z9fekv4etlcks5qwdy3pz")
		if err != nil {
			t.Fatal(err)
		}
		require.Equal("DWIN", symbol)
	})
	t.Run("ReadERC20Name", func(t *testing.T) {
		name, err := ReadERC20Name(client, "io15uggvd649nk8arpd6z9fekv4etlcks5qwdy3pz")
		if err != nil {
			t.Fatal(err)
		}
		require.Equal("Drop Wireless INfrastructure", name)
	})
}
