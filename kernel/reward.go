package kernel

import (
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/rewardingpb"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
)

type RewardInfo struct {
	BlockReward       *big.Int
	EpochReward       *big.Int
	FoundationBonus   *big.Int
	PriorityBonus     *big.Int
	UnproductiveSlash *big.Int
}

func RewardInfoFromReceipt(receipt *action.Receipt) (map[string]*RewardInfo, error) {
	rewardInfoMap := make(map[string]*RewardInfo)
	for _, l := range receipt.Logs() {
		logs, err := rewarding.UnmarshalRewardLog(l.Data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal receipt data into reward log")
		}
		for _, log := range logs.Logs {
			rewardLog := log
			rewards, ok := rewardInfoMap[rewardLog.Addr]
			if !ok {
				rewardInfoMap[rewardLog.Addr] = &RewardInfo{
					BlockReward:       big.NewInt(0),
					EpochReward:       big.NewInt(0),
					FoundationBonus:   big.NewInt(0),
					PriorityBonus:     big.NewInt(0),
					UnproductiveSlash: big.NewInt(0),
				}
				rewards = rewardInfoMap[rewardLog.Addr]
			}
			amount, ok := big.NewInt(0).SetString(rewardLog.Amount, 10)
			if !ok {
				return nil, errors.Errorf("failed to convert reward amount from string to big int: %s, %s, %s", rewardLog.Addr, rewardLog.Type, rewardLog.Amount)
			}
			switch rewardLog.Type {
			case rewardingpb.RewardLog_BLOCK_REWARD:
				rewards.BlockReward.Add(rewards.BlockReward, amount)
			case rewardingpb.RewardLog_EPOCH_REWARD:
				rewards.EpochReward.Add(rewards.EpochReward, amount)
			case rewardingpb.RewardLog_FOUNDATION_BONUS:
				rewards.FoundationBonus.Add(rewards.FoundationBonus, amount)
			case rewardingpb.RewardLog_PRIORITY_BONUS:
				rewards.PriorityBonus.Add(rewards.PriorityBonus, amount)
			case rewardingpb.RewardLog_UNPRODUCTIVE_SLASH:
				rewards.UnproductiveSlash.Add(rewards.UnproductiveSlash, amount)
			default:
				return nil, errors.New("Unknown type of reward")
			}
		}
	}
	return rewardInfoMap, nil
}

// RewardAt returns all kinds of rewards in a block
// TODO: gasConsumed should not be returned here
func RewardAt(blk *block.Block, grantRewardActs map[hash.Hash256]bool) (*big.Int, *big.Int, *big.Int, *big.Int, uint64, error) {
	blockReward := big.NewInt(0)
	epochReward := big.NewInt(0)
	foundationBonus := big.NewInt(0)
	priorityBonus := big.NewInt(0)
	var gasConsumed uint64
	// log receipt index
	for _, receipt := range blk.Receipts {
		gasConsumed += receipt.GasConsumed
		if _, ok := grantRewardActs[receipt.ActionHash]; ok {
			// Parse receipt of grant reward
			rewardInfoMap, err := RewardInfoFromReceipt(receipt)
			if err != nil {
				return blockReward, epochReward, foundationBonus, priorityBonus, gasConsumed, errors.Wrap(err, "failed to get reward info from receipt")
			}
			if len(rewardInfoMap) == 0 {
				continue
			}
			for _, rewards := range rewardInfoMap {
				blockReward.Add(blockReward, rewards.BlockReward)
				epochReward.Add(epochReward, rewards.EpochReward)
				foundationBonus.Add(foundationBonus, rewards.FoundationBonus)
				priorityBonus.Add(priorityBonus, rewards.PriorityBonus)
			}
		}
	}
	return blockReward, epochReward, foundationBonus, priorityBonus, gasConsumed, nil
}
