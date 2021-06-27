package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.0.0"

type accountVotePlugin struct {
}

func (b accountVotePlugin) Name() string {
	return "account_vote"
}

func (b accountVotePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b accountVotePlugin) Start(ctx context.Context) error {
	if err := db.DB().AutoMigrate(&AccountVote{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	var err error
	config, _ := kernel.GetConfigCtx(ctx)
	_, err = newConfig(config)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s plugin config", b.Name())
	}
	return nil
}

func (b accountVotePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	// for _, receipt := range blk.Receipts {
	// 	for _, logs := range receipt.Logs() {
	// 		logs.
	// 	}
	// }
	zeroInt := big.NewInt(0)
	zeroDecimal := decimal.NewFromBigInt(zeroInt, 0)
	accountVotes := []AccountVote{}
	for _, selp := range blk.Actions {
		sender, _ := address.FromBytes(selp.SrcPubkey().Hash())
		act := selp.Action()
		switch a := act.(type) {
		case *action.CreateStake:
			cadidateAddr, err := getCandidateAddressByName(a.Candidate())
			if err != nil {
				return err
			}
			actionHash := selp.Hash()
			bucketID, err := getBucketIDByActHash(hex.EncodeToString(actionHash[:]))
			if err != nil {
				return err
			}
			voteBucket := &VoteBucket{
				Index:          bucketID,
				Candidate:      cadidateAddr,
				Owner:          sender.String(),
				AutoStake:      a.AutoStake(),
				StakedAmount:   a.Amount(),
				StakedDuration: time.Duration(a.Duration()),
			}
			selfStake := false
			if cadidateAddr == sender.String() {
				selfStake = true
			}
			voteWeight := calculateVoteWeight(Default.VoteWeightCalConsts, voteBucket, selfStake)
			accountVotes = append(accountVotes, AccountVote{
				BlockHeight:           blk.Height(),
				BucketID:              bucketID,
				Address:               sender.String(),
				Candidate:             cadidateAddr,
				CreateStakeAmount:     decimal.NewFromBigInt(a.Amount(), 0),
				UnStakeAmount:         zeroDecimal,
				CreateStakeVoteWeight: decimal.NewFromBigInt(voteWeight, 0),
				UnStakeVoteWeight:     zeroDecimal,
			})
		case *action.Unstake:
			bucketID := a.BucketIndex()
			var av AccountVote
			if err := db.DB().Model(&AccountVote{}).Where("bucket_id=? and address=?", bucketID, sender.String()).First(&av).Error; err != nil {
				return err
			}
			accountVotes = append(accountVotes, AccountVote{
				BlockHeight:           blk.Height(),
				BucketID:              bucketID,
				Address:               sender.String(),
				Candidate:             av.Candidate,
				CreateStakeAmount:     zeroDecimal,
				UnStakeAmount:         av.CreateStakeAmount,
				CreateStakeVoteWeight: zeroDecimal,
				UnStakeVoteWeight:     av.CreateStakeVoteWeight,
			})
		default:
			continue
		}
	}
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, accountVote := range accountVotes {
			if err := tx.Create(&accountVote).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return err
}

func (b accountVotePlugin) Stop(ctx context.Context) error {
	return nil
}

func (b accountVotePlugin) Version() string {
	return VERSION
}

// exported
var Plugin = accountVotePlugin{}
