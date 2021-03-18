package cmd

import (
	"github.com/iotexproject/iotex-analyser/cmd/plugin"
	"github.com/urfave/cli/v2"
)

var Plugin = &cli.Command{
	Name:  "plugin",
	Usage: "load/unload plugin",
	Subcommands: []*cli.Command{
		plugin.Load,
		plugin.UnLoad,
		plugin.Info,
	},
}
