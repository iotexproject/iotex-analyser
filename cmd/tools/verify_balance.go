package tools

import (
	"database/sql"
	"fmt"
	"log"
	"runtime"

	"github.com/gammazero/workerpool"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/urfave/cli/v2"
)

var VerifyBalance = &cli.Command{
	Name:        "verifybalance",
	Usage:       "verifybalance -min <minheight> -max <maxheight> -worker <workersize>",
	Description: `verify iotex account balance`,
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
	Action: verifyBalance,
}

func verifyBalance(c *cli.Context) error {
	fmt.Printf("min=%d max=%d worker=%d\n", c.Uint64("min"), c.Uint64("max"), c.Int("worker"))
	wp := workerpool.New(c.Int("worker"))

	// for i := c.Uint64("min"); i < c.Uint64("max"); i++ {
	// 	blkHeight := i
	// 	wp.Submit(func() {
	// 		verifyBalanceWorker(blkHeight)
	// 	})
	// }
	db := kernel.GetDB()
	query := `select DISTINCT account_address from account_income where block_height>=? and block_height<?`
	rows, err := db.Query(query, c.Uint64("min"), c.Uint64("max"))
	if err != nil {
		return err
	}

	defer rows.Close()

	var addr sql.NullString
	for rows.Next() {
		if err = rows.Scan(
			&addr,
		); err != nil {
			return err
		}
		address := addr.String
		if address == "io0000000000000000000000rewardingprotocol" ||
			address == "" {
			continue
		}
		wp.Submit(func() {
			verifyBalanceWorker(address, c)
		})
	}
	wp.StopWait()
	fmt.Println("done")
	return nil
}

func verifyBalanceWorker(addr string, c *cli.Context) {
	var err error
	db := kernel.GetDB()
	min := c.Uint64("min")
	max := c.Uint64("max")
	query := `select (SELECT IFNULL(sum(amount),0) FROM block_receipt_transaction WHERE  block_height>=? and block_height<? and recipient=?)- (SELECT IFNULL(sum(amount),0) FROM block_receipt_transaction WHERE block_height>=? and block_height<? and sender=?)`
	var amount1 sql.NullString
	err = db.QueryRow(query, min, max, addr, min, max, addr).Scan(&amount1)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("%s got error %v", addr, err)
		return
	}
	query = `SELECT sum(in_flow)-sum(out_flow) FROM account_income WHERE account_address=? and block_height>=? and block_height<?`
	var amount2 sql.NullString
	err = db.QueryRow(query, addr, min, max).Scan(&amount2)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("%s got error %v", addr, err)
		return
	}
	if amount1.String != amount2.String {
		fmt.Println(addr, amount1.String, " != ", amount2.String)
	}

	/*
			select (SELECT sum(amount) FROM block_receipt_transaction WHERE  block_height<=36000 and recipient='io17ch0jth3dxqa7w9vu05yu86mqh0n6502d92lmp')- (SELECT sum(amount) FROM block_receipt_transaction WHERE block_height<=36000 and sender='io17ch0jth3dxqa7w9vu05yu86mqh0n6502d92lmp')

		SELECT sum(in_flow)-sum(out_flow) FROM `account_income` WHERE `account_address` = 'io17ch0jth3dxqa7w9vu05yu86mqh0n6502d92lmp' and block_height<=36000;
	*/

}
