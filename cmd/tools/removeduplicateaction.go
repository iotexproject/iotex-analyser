package tools

import (
	"fmt"
	"os"
	"runtime"

	"github.com/gammazero/workerpool"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/pkg/errors"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

var RemoveDuplicateActions = &cli.Command{
	Name:        "remove_duplicate_actions",
	Usage:       "remove_duplicate_actions --start=<startBlkNum> --end=<endBlkNum> --worker <workerSize>",
	Description: "remove_duplicate_actions will remove duplicate actions from database",
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
		&cli.IntFlag{
			Name:  "worker",
			Usage: "worker thread num",
			Value: runtime.NumCPU(),
		},
	},
	Action: removeDuplicateActions,
}

func removeDuplicateActions(c *cli.Context) error {
	startBlkNum := c.Uint64("start")
	endBlkNum := c.Uint64("end")
	db, err := db.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect db")
	}
	if startBlkNum > endBlkNum {
		return errors.New("start blkNum is bigger than end blkNum")
	}

	wp := workerpool.New(c.Int("worker"))
	bar := progressbar.NewOptions(int(endBlkNum-startBlkNum+1),
		progressbar.OptionSetDescription("Verifying..."),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
	)
	for i := startBlkNum; i <= endBlkNum; i++ {
		i := i
		wp.Submit(func() {
			defer bar.Add(1)
			if err := removeAction(db, i); err != nil {
				fmt.Printf("\ndelete %d error=%v \n", i, err)
			}

		})
	}
	wp.StopWait()
	return nil
}

func removeAction(db *gorm.DB, height uint64) error {
	return db.Exec(`delete from block_action where id in(select id from (SELECT id,ROW_NUMBER() OVER(PARTITION BY action_hash,action_type,sender,recipient,gas_price,gas_limit,nonce,amount,gas_consumed,chain_id,"encoding","version",contract_address,status,execution_revert_msg,payload ORDER BY id asc) AS Row FROM block_action where  block_height=? ) dups where dups.Row > 1)`, height).Error
}
