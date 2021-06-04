package tools

import (
	"database/sql"
	"errors"
	"fmt"
	"runtime"

	"github.com/cheggaaa/pb/v3"
	"github.com/gammazero/workerpool"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

var FixAccountIncome = &cli.Command{
	Name:        "fixAccountIncome",
	Usage:       "fixAccountIncome --block <height> --worker <threads>",
	Description: `fix account_income table`,
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "block",
			Usage: "block height",
		},
		&cli.IntFlag{
			Name:  "worker",
			Usage: "worker thread num",
			Value: runtime.NumCPU() * 2,
		},
	},
	Action: fixAccountIncome,
}

func fixAccountIncome(c *cli.Context) error {
	blkHeight := c.Uint64("block")
	if blkHeight == 0 {
		return errors.New("--block must > 0")
	}
	db := db.DB()

	var height sql.NullInt64
	query := "SELECT height FROM index_heights WHERE name='account_income'"
	db.Raw(query).Scan(&height)
	if height.Int64 == 0 {
		return nil
	}
	count := int(height.Int64 - int64(blkHeight))
	bar = pb.StartNew(count)
	wp := workerpool.New(c.Int("worker"))
	for i := uint64(height.Int64); i >= blkHeight; i-- {
		i := i
		wp.Submit(func() {
			deleteAccountIncomeRow(i)
		})

	}
	wp.StopWait()
	bar.Finish()
	fmt.Println("rebuilding account_income_count table")
	err := db.Transaction(func(tx *gorm.DB) error {
		err := tx.Exec("truncate account_income_count").Error
		if err != nil {
			return err
		}
		err = tx.Exec("insert into account_income_count(account_address,in_flow,in_num_actions,out_flow,out_num_actions) select account_address,SUM(in_flow),SUM(in_num_actions),SUM(out_flow),SUM(out_num_actions) from account_income GROUP BY account_address").Error
		if err != nil {
			return err
		}
		err = tx.Exec("update index_heights set height=? where name='account_income'", blkHeight-1).Error
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func deleteAccountIncomeRow(blkHeight uint64) {
	bar.Increment()
	err := db.DB().Transaction(func(t *gorm.DB) error {
		err := t.Exec("DELETE FROM account_income WHERE block_height=?", blkHeight).Error
		return err
	})
	if err != nil {
		panic(err)
	}
}
