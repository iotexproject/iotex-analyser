package tools

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"

	"github.com/cheggaaa/pb/v3"
	"github.com/gammazero/workerpool"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-core/action/protocol"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

var VerifyBlockReceipt = &cli.Command{
	Name:        "verifyBlockReceipt",
	Usage:       "verifyBlockReceipt --min <minheight> --max <maxheight> --worker <workersize>",
	Description: `verify BlockReceipt`,
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "min",
			Usage: "min block height",
			Value: 1,
		},
		&cli.Uint64Flag{
			Name:  "max",
			Usage: "max block height",
			Value: 10000,
		},
		&cli.IntFlag{
			Name:  "worker",
			Usage: "worker thread num",
			Value: runtime.NumCPU() * 2,
		},
	},
	Action: verifyBlockReceipt,
}

func verifyBlockReceipt(c *cli.Context) error {
	fmt.Printf("min=%d max=%d worker=%d\n", c.Uint64("min"), c.Uint64("max"), c.Int("worker"))
	db, err := db.Connect()
	if err != nil {
		return err
	}
	min := c.Uint64("min")
	max := c.Uint64("max")

	var height sql.NullInt64
	query := "SELECT height FROM index_heights WHERE name='block_receipt'"
	if err := db.Raw(query).Scan(&height).Error; err != nil {
		return errors.Wrap(err, "")
	}
	if height.Int64 == 0 {
		return nil
	}

	if uint64(height.Int64) < max {
		max = uint64(height.Int64)
	}
	var tip protocol.TipInfo
	ctxDao := protocol.WithBlockchainCtx(
		context.Background(),
		protocol.BlockchainCtx{
			Genesis: genesis.Default,
			Tip:     tip,
		},
	)
	var indexers []blockdao.BlockIndexer
	var dao blockdao.BlockDAO
	dao = blockdao.NewBlockDAO(indexers, config.Default.BlockDB)
	if err := dao.Start(ctxDao); err != nil {
		return err
	}
	count := int(max - min)
	bar = pb.StartNew(count)
	wp := workerpool.New(c.Int("worker"))

	for i := min; i <= max; i++ {
		i := i
		wp.Submit(func() {
			defer bar.Increment()
			tlogs, err := dao.TransactionLogs(i)
			if err != nil {
				panic(err)
			}
			tlogsLen := 0
			for _, l := range tlogs.Logs {
				tlogsLen += len(l.Transactions)
			}
			if tlogsLen == 0 {
				return
			}
			var count sql.NullInt32
			query := `select count(1) from block_receipt_transaction where block_height=?`
			if err := db.Raw(query, i).Scan(&count).Error; err != nil && err != gorm.ErrRecordNotFound {
				panic(err)
			}
			if int(count.Int32) != tlogsLen {
				fmt.Printf("height: %d, logsLen: %d, dbLen: %d\n", i, tlogsLen, count.Int32)
			}
		})

	}
	wp.StopWait()
	bar.Finish()
	return dao.Stop(ctxDao)
}
