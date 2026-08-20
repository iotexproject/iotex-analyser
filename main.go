package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/iotexproject/iotex-analyser/cmd"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/kernel"
)

var (
	version = "1.3.1"
)

func main() {
	// Pin the process to UTC before anything can read a clock.
	//
	// Block timestamps are written to `timestamp without time zone` columns via
	// time.Unix(), which returns a time in the *local* zone; the driver then
	// stores that zone's wall-clock reading. On a UTC host that is correct by
	// accident. On a UTC+8 host every block, action and transfer lands eight
	// hours in the future, and the explorer renders freshly mined blocks as
	// "in 7 hours".
	//
	// Fixing the ~68 individual call sites would leave the next one to be
	// written wrong, so the timezone is removed as a variable instead. An
	// indexer has no business rendering local time anyway.
	time.Local = time.UTC

	app := cli.NewApp()
	app.Name = "iotex-analyser"
	app.Usage = "async analyser for iotex"
	app.Version = version
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Value:   config.FindDefaultConfigPath(),
			Usage:   "Load configuration from `FILE`",
		},
	}
	app.Before = func(c *cli.Context) error {
		if c.String("config") == "" {
			log.L().Fatal("Cannot determine default configuration path.",
				zap.Any("DefaultConfigFiles", config.DefaultConfigFiles),
				zap.Any("DefaultConfigDirs", config.DefaultConfigDirs))
		}

		cfg, err := config.New(c.String("config"))
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

		log.L().Debug("loaded iotex-core configure",
			zap.String("version", version),
			zap.Uint32("EVMNetworkID", config.EVMNetworkID()),
		)
		kernel.Init(cfg)
		return nil
	}
	app.Commands = []*cli.Command{
		cmd.Server,
		cmd.Plugin,
		cmd.Tools,
	}
	if err := app.Run(os.Args); err != nil {
		log.L().Error("Failed to start application", zap.Error(err))
	}
}
