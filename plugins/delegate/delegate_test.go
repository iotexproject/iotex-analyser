package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestDelegate(t *testing.T) {
	r := require.New(t)
	err := Init("../../.vscode/mainnet.config.yml")
	r.NoError(err)
	delegate()
}

func Init(path string) error {
	cfg, err := config.New(path)
	if err != nil {
		return err
	}
	if err := log.InitLoggers(cfg.Log, cfg.SubLogs); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	config.SetEVMNetworkID(cfg.Iotex.EVMNetworkID)
	if cfg.Iotex.EVMNetworkID == 0 {
		if strings.Contains(cfg.Iotex.ChainEndPoint, "testnet") {
			config.SetEVMNetworkID(4690)
		} else {
			config.SetEVMNetworkID(4689)
		}
	}

	kernel.Init(cfg)

	_, err = db.Connect()
	return err
}
