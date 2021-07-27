package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.0.0"

var nextHeight uint64

type airdripPlugin struct {
}

func (b airdripPlugin) Name() string {
	return "airdrip"
}

func (b airdripPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b airdripPlugin) Start(ctx context.Context) error {
	config, _ := kernel.GetConfigCtx(ctx)
	cfg, err := newConfig(config)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s plugin config", b.Name())
	}

	if cfg.Airdrip.InitHeight == 0 {
		return errors.New("please set airdrip.initHeight in config")
	}
	res, err := getAirdripFromStore()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			currentHeight, err := db.GetIndexHeight(b.Name())
			if err != nil {
				return err
			}
			js := &storeJson{
				CurrentHeight: currentHeight,
				NextHeight:    calcNextHeight(currentHeight),
				UpdateTime:    time.Now(),
			}
			raw, _ := json.Marshal(js)

			store := &db.Store{
				Key:   StoreKey,
				Value: string(raw),
			}
			if err := store.Save(); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	nextHeight = res.NextHeight
	return nil
}

func (b airdripPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	//skip unreached height
	if nextHeight != blkHeight {
		return db.UpdateIndexHeight(b.Name(), blk.Height())
	}
	buckets, err := getAliveBucketIDs(blkHeight)
	if err != nil {
		return err
	}
	ownerVotes := make(map[string]OwnerVote)
	for _, bucketID := range buckets {
		ownerAddr, err := getBucketOwnerWithHeight(bucketID, blkHeight)
		if err != nil {
			return err
		}
		ownerVote, ok := ownerVotes[ownerAddr]
		if !ok {
			ownerVote = OwnerVote{
				StakeAmount: big.NewInt(0),
				VoteWeight:  big.NewInt(0),
			}
		}

		stakeAmount, err := getSumStake(ownerAddr, blkHeight, bucketID)
		if err != nil {
			return err
		}
		ownerVote.StakeAmount = ownerVote.StakeAmount.Add(ownerVote.StakeAmount, stakeAmount)
		duration, autoStake, selfAutoStake := getVoteBucketParams(ownerAddr, blkHeight, bucketID)
		voteBucket := &VoteBucket{
			StakedAmount:   stakeAmount,
			AutoStake:      autoStake,
			StakedDuration: duration,
		}
		voteWeight := calculateVoteWeight(Default.Genesis.VoteWeightCalConsts, voteBucket, selfAutoStake)
		ownerVote.VoteWeight = ownerVote.VoteWeight.Add(ownerVote.VoteWeight, voteWeight)
		ownerVotes[ownerAddr] = ownerVote
	}

	users := make([][32]byte, len(ownerVotes))
	shares := make([]*big.Int, len(ownerVotes))
	for addr, vote := range ownerVotes {
		evmAddr, err := ioAddrToEvmAddr(addr)
		if err != nil {
			return err
		}
		users = append(users, stringToBytes32(evmAddr.String()))
		shares = append(shares, vote.VoteWeight)
		fmt.Printf("%s => %s  %s\n", addr, vote.StakeAmount, vote.VoteWeight)
	}
	fmt.Printf("%v => %v\n", users, shares)
	nextHeight = calcNextHeight(blkHeight)
	raw, _ := json.Marshal(&storeJson{
		CurrentHeight: blkHeight,
		NextHeight:    nextHeight,
		UpdateTime:    time.Now(),
	})

	store := &db.Store{
		Key:   StoreKey,
		Value: string(raw),
	}
	if err := store.Save(); err != nil {
		return err
	}
	return nil
}

func (b airdripPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b airdripPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = airdripPlugin{}
