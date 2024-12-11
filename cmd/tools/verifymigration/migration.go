package verifymigration

import (
	"github.com/urfave/cli/v2"
)

var VerifyMigration = &cli.Command{
	Name:  "verify_migration",
	Usage: "verify_migration <subcommand>",
	Subcommands: []*cli.Command{
		VerifyActionExecutionCmd,
	},
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "start",
			Usage: "start block number",
			Value: 1,
		},
		&cli.Uint64Flag{
			Name:  "end",
			Usage: "end block number",
			Value: 1,
		},
		&cli.StringFlag{
			Name:  "chdsn",
			Usage: "clickhouse DSN",
			Value: "clickhouse://username:password@127.0.0.1:9000/testnet",
		},
		&cli.StringFlag{
			Name:  "pgdsn",
			Usage: "postgres DSN",
			Value: "postgres://postgres:mysecretpassword@127.0.0.1:5432/testnet?sslmode=disable",
		},
	},
}
