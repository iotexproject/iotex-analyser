package tools

import (
	"database/sql"
	"strconv"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var FixBlockReceipt = &cli.Command{
	Name:        "fixBlockReceipt",
	Usage:       "fixBlockReceipt --block <height>",
	Description: `fix block receipt data`,
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "block",
			Usage: "block height",
		},
	},
	Action: fixBlockReceipt,
}

func getRowID(t *sql.Tx, query string, args ...interface{}) int64 {
	var id sql.NullInt64
	t.QueryRow(query, args...).Scan(&id)
	return id.Int64
}
func fixBlockReceipt(c *cli.Context) error {
	blkHeight := c.Uint64("block")
	if blkHeight == 0 {
		return errors.New("missing --block <height>")
	}

	db := kernel.GetDB()
	defer db.Close()
	err := kernel.Transaction(func(t *sql.Tx) error {
		var err error
		var id int64
		id = getRowID(t, "select max(id) from block_receipt where block_height<?", blkHeight)
		if id > 0 {
			_, err = t.Exec("DELETE FROM block_receipt WHERE id>?", id)
			if err != nil {
				return errors.Wrap(err, "")
			}
			_, err = t.Exec("alter table block_receipt auto_increment=" + strconv.FormatUint(uint64(id+1), 10))
			if err != nil {
				return errors.Wrap(err, "")
			}
		}

		id = getRowID(t, "select max(id) from block_receipt_log where block_height<?", blkHeight)
		if id > 0 {
			_, err = t.Exec("DELETE FROM block_receipt_log WHERE id>?", id)
			if err != nil {
				return errors.Wrap(err, "")
			}
			_, err = t.Exec("alter table block_receipt_log auto_increment=" + strconv.FormatUint(uint64(id+1), 10))
			if err != nil {
				return errors.Wrap(err, "")
			}
		}
		id = getRowID(t, "select max(id) from block_receipt_transaction where block_height<?", blkHeight)
		if id > 0 {
			_, err = t.Exec("DELETE FROM block_receipt_transaction WHERE id>?", id)
			if err != nil {
				return errors.Wrap(err, "")
			}
			_, err = t.Exec("alter table block_receipt_transaction auto_increment=" + strconv.FormatUint(uint64(id+1), 10))
			if err != nil {
				return errors.Wrap(err, "")
			}
		}
		_, err = t.Exec("update index_heights set height=? where name='block_receipt'", blkHeight-1)
		if err != nil {
			return errors.Wrap(err, "")
		}
		return err
	})
	return err
}
