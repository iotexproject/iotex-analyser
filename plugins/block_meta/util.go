package main

import (
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
)

func getCandidateName(height uint64, address string) string {
	var cand models.Candidate
	var name string
	err := db.DB().Model(&cand).Where("block_height <=? and operator_address = ?", height, address).Order("id desc").Take(&cand).Error
	if err == nil {
		name = cand.Name
	}
	return name
}

func getReward(blk *block.Block, grantRewardActs map[hash.Hash256]bool) (*big.Int, *big.Int, *big.Int, uint64, error) {
	blockReward, epochReward, foundationBonus, _, gasConsumed, err := kernel.RewardAt(blk, grantRewardActs)
	return blockReward, epochReward, foundationBonus, gasConsumed, err
}

func getBlockSize(blk *block.Block) (uint64, error) {
	size := uint64(0)
	//block data
	blkInfo := &block.Store{
		Block:    blk,
		Receipts: blk.Receipts,
	}
	ser, err := blkInfo.Serialize()
	if err != nil {
		return 0, err
	}
	size += uint64(len(ser))

	//receipt and transaction log
	sysLog := blk.TransactionLog()
	if sysLog == nil {
		sysLog = &block.BlkTransactionLog{}
	}
	size += uint64(len(sysLog.Serialize()))
	return size, nil
}
