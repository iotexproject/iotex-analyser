package cmd

import (
	"github.com/iotexproject/iotex-analyser/cmd/tools"
	"github.com/iotexproject/iotex-analyser/cmd/tools/verifymigration"
	"github.com/urfave/cli/v2"
)

var Tools = &cli.Command{
	Name:  "tools",
	Usage: "tools <subcommand>",
	Subcommands: []*cli.Command{
		tools.VerifyDB,
		verifymigration.VerifyMigration,
	},
}
