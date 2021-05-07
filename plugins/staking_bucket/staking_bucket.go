package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
)

const VERSION = "1.0.0"

const successStatus = uint64(1)

type stakingBucketPlugin struct {
	tableName string
}

func (b stakingBucketPlugin) Name() string {
	return "staking_bucket"
}

func (b stakingBucketPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b stakingBucketPlugin) Start(ctx context.Context) error {
	createSql := "CREATE TABLE IF NOT EXISTS `" + b.tableName + "` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`action_hash` varchar(64) NOT NULL DEFAULT ''," +
		"`bucket_id` DECIMAL(42, 0) UNSIGNED NOT NULL DEFAULT 0," +
		"PRIMARY KEY (`id`)," +
		"KEY `action_hash` (`action_hash`(9))," +
		"KEY `bucket_id` (`bucket_id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b stakingBucketPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	h := hash.Hash160b([]byte("staking"))
	stakingProtocolAddr, err := address.FromBytes(h[:])
	if err != nil {
		return err
	}
	//io1qnpz47hx5q6r3w876axtrn6yz95d70cjl35r53

	err = kernel.Transaction(func(tx *sql.Tx) error {

		for _, receipt := range blk.Receipts {
			if receipt.Status != successStatus {
				continue
			}
			actionHash := hex.EncodeToString(receipt.ActionHash[:])
			for _, log := range receipt.Logs() {
				if log.Address == stakingProtocolAddr.String() && len(log.Topics) > 1 {
					bucketIndex := new(big.Int).SetBytes(log.Topics[1][:])

					insertData := map[string]interface{}{
						"action_hash": actionHash,
						"bucket_id":   bucketIndex.String(),
					}
					if err := kernel.InsertTableData(tx, b.tableName, insertData); err != nil {
						return errors.Wrap(err, "failed to insert table data")
					}
				}
			}
		}

		return kernel.UpdateIndexHeight(tx, b.Name(), blk.Height())
	})

	return err
}

func (b stakingBucketPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b stakingBucketPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = stakingBucketPlugin{
	tableName: "staking_bucket",
}
