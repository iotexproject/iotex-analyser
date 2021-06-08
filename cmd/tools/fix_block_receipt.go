package tools

import (
	"database/sql"
	"strconv"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
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

func getRowID(t *gorm.DB, query string, args ...interface{}) int64 {
	var id sql.NullInt64
	t.Raw(query, args...).Scan(&id)
	return id.Int64
}
func fixBlockReceipt(c *cli.Context) error {
	blkHeight := c.Uint64("block")
	if blkHeight == 0 {
		return errors.New("missing --block <height>")
	}

	db, err := db.Connect()
	if err != nil {
		return err
	}
	err = db.Transaction(func(t *gorm.DB) error {
		var err error
		var id int64
		var minBlkHeight uint64
		if blkHeight < 1000 {
			minBlkHeight = 0
		} else {
			minBlkHeight = blkHeight - 1000
		}
		id = getRowID(t, "select max(id) from block_receipt where block_height>? and block_height<?", minBlkHeight, blkHeight)
		if id > 0 {
			err = t.Exec("DELETE FROM block_receipt WHERE id>?", id).Error
			if err != nil {
				return errors.Wrap(err, "")
			}
			err = t.Exec("alter sequence block_receipt_id_seq restart with " + strconv.FormatUint(uint64(id), 10)).Error
			if err != nil {
				return errors.Wrap(err, "")
			}
		}

		id = getRowID(t, "select max(id) from block_receipt_log where block_height>? and block_height<?", minBlkHeight, blkHeight)
		if id > 0 {
			err = t.Exec("DELETE FROM block_receipt_log WHERE id>?", id).Error
			if err != nil {
				return errors.Wrap(err, "")
			}
			err = t.Exec("alter sequence block_receipt_log_id_seq restart with " + strconv.FormatUint(uint64(id), 10)).Error
			if err != nil {
				return errors.Wrap(err, "")
			}
		}
		id = getRowID(t, "select max(id) from block_receipt_transaction where block_height>? and block_height<?", minBlkHeight, blkHeight)
		if id > 0 {
			err = t.Exec("DELETE FROM block_receipt_transaction WHERE id>?", id).Error
			if err != nil {
				return errors.Wrap(err, "")
			}
			err = t.Exec("alter sequence block_receipt_transaction_id_seq restart with " + strconv.FormatUint(uint64(id), 10)).Error
			if err != nil {
				return errors.Wrap(err, "")
			}
		}
		err = t.Exec("update index_heights set height=? where name='block_receipt'", blkHeight-1).Error
		if err != nil {
			return errors.Wrap(err, "")
		}
		return err
	})
	return err
}
