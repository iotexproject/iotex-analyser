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

// getCandidateNamesBatch fetches the most-recent name for each address in addrs
// as of the given block height, in a single query. Returns a map of
// operator_address → name. Addresses with no record are omitted from the map.
func getCandidateNamesBatch(height uint64, addrs []string) (map[string]string, error) {
	if len(addrs) == 0 {
		return map[string]string{}, nil
	}
	type row struct {
		OperatorAddress string
		Name            string
	}
	// DISTINCT ON keeps the row with the highest id (most recent) per operator_address.
	var rows []row
	err := db.DB().Raw(`
		SELECT DISTINCT ON (operator_address) operator_address, name
		FROM candidate
		WHERE block_height <= ? AND operator_address IN ?
		ORDER BY operator_address, id DESC`,
		height, addrs,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(rows))
	for _, r := range rows {
		result[r.OperatorAddress] = r.Name
	}
	return result, nil
}

func getReward(blk *block.Block, grantRewardActs map[hash.Hash256]bool) (*big.Int, *big.Int, *big.Int, *big.Int, uint64, error) {
	return kernel.RewardAt(blk, grantRewardActs)
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
