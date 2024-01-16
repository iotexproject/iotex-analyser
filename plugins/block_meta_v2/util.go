package main

import (
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/action/protocol/rewarding/rewardingpb"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type RewardInfo struct {
	BlockReward     *big.Int
	EpochReward     *big.Int
	FoundationBonus *big.Int
}

func getRewardInfoFromReceipt(receipt *action.Receipt) (map[string]*RewardInfo, error) {
	rewardInfoMap := make(map[string]*RewardInfo)
	for _, l := range receipt.Logs() {
		rewardLog := &rewardingpb.RewardLog{}
		if err := proto.Unmarshal(l.Data, rewardLog); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal receipt data into reward log")
		}
		rewards, ok := rewardInfoMap[rewardLog.Addr]
		if !ok {
			rewardInfoMap[rewardLog.Addr] = &RewardInfo{
				BlockReward:     big.NewInt(0),
				EpochReward:     big.NewInt(0),
				FoundationBonus: big.NewInt(0),
			}
			rewards = rewardInfoMap[rewardLog.Addr]
		}
		amount, ok := big.NewInt(0).SetString(rewardLog.Amount, 10)
		if !ok {
			return nil, errors.New("failed to convert reward amount from string to big int")
		}
		switch rewardLog.Type {
		case rewardingpb.RewardLog_BLOCK_REWARD:
			rewards.BlockReward.Add(rewards.BlockReward, amount)
		case rewardingpb.RewardLog_EPOCH_REWARD:
			rewards.EpochReward.Add(rewards.EpochReward, amount)
		case rewardingpb.RewardLog_FOUNDATION_BONUS:
			rewards.FoundationBonus.Add(rewards.FoundationBonus, amount)
		default:
			return nil, errors.New("Unknown type of reward")
		}
	}
	return rewardInfoMap, nil
}

func getCandidateName(height uint64, address string) (string, string) {
	var cand models.Candidate
	var name, ownerAddress string
	err := db.DB().Model(&cand).Where("block_height <=? and operator_address = ?", height, address).Order("id desc").Take(&cand).Error
	if err == nil {
		name = cand.Name
		ownerAddress = cand.OwnerAddress
	}
	return name, ownerAddress
}

func getReward(blk *block.Block, grantRewardActs map[hash.Hash256]bool) (*big.Int, *big.Int, *big.Int, uint64, error) {
	blockReward := big.NewInt(0)
	epochReward := big.NewInt(0)
	foundationBonus := big.NewInt(0)
	var gasConsumed uint64
	// log receipt index
	for _, receipt := range blk.Receipts {
		gasConsumed += receipt.GasConsumed
		if _, ok := grantRewardActs[receipt.ActionHash]; ok {
			// Parse receipt of grant reward
			rewardInfoMap, err := getRewardInfoFromReceipt(receipt)
			if err != nil {
				return blockReward, epochReward, foundationBonus, gasConsumed, errors.Wrap(err, "failed to get reward info from receipt")
			}
			if len(rewardInfoMap) == 0 {
				continue
			}
			for _, rewards := range rewardInfoMap {
				blockReward.Add(blockReward, rewards.BlockReward)
				epochReward.Add(epochReward, rewards.EpochReward)
				foundationBonus.Add(foundationBonus, rewards.FoundationBonus)
			}
		}
	}
	return blockReward, epochReward, foundationBonus, gasConsumed, nil
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
