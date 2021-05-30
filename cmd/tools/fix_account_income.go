package tools

import (
	"errors"

	"github.com/urfave/cli/v2"
)

var FixAccountIncome = &cli.Command{
	Name:        "fixAccountIncome",
	Usage:       "fixAccountIncome --block <height>",
	Description: `fix account_income table`,
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "block",
			Usage: "block height",
		},
	},
	Action: fixAccountIncome,
}

func fixAccountIncome(c *cli.Context) error {
	blkHeight := c.Uint64("block")
	if blkHeight == 0 {
		return errors.New("--block must > 0")
	}

	return nil
}
