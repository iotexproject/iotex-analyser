package main

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.0.1"

const successStatus = uint64(1)

type stakingBucketPlugin struct {
}

func (b stakingBucketPlugin) Name() string {
	return "staking_bucket"
}

func (b stakingBucketPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b stakingBucketPlugin) Start(ctx context.Context) error {
	if err := db.DB().AutoMigrate(&StakingBucket{}); err != nil {
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

	err = db.DB().Transaction(func(tx *gorm.DB) error {
		cmpNum := big.NewInt(100000000)

		for _, receipt := range blk.Receipts {
			if receipt.Status != successStatus {
				continue
			}
			actionHash := hex.EncodeToString(receipt.ActionHash[:])
			for _, log := range receipt.Logs() {
				if log.Address == stakingProtocolAddr.String() && len(log.Topics) > 1 {
					bucketIndex := new(big.Int).SetBytes(log.Topics[1][:])

					if bucketIndex.Cmp(cmpNum) > 0 {
						continue
					}
					bucketID := decimal.NewFromBigInt(bucketIndex, 0)
					m := &StakingBucket{
						ActionHash: actionHash,
						BucketID:   bucketID,
					}
					if err := tx.Create(m).Error; err != nil {
						return errors.Wrap(err, "failed to insert table data")
					}
				}
			}
		}

		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
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
var Plugin = stakingBucketPlugin{}
