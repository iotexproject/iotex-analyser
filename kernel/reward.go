package kernel

import (
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/rewardingpb"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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
		if len(l.Topics) != 0 {
			// Not a RewardLog. Before Zanzibar a GrantReward receipt carried
			// nothing else, so unmarshalling every log unconditionally was
			// safe. IIP-59 now emits DelegateVoterRewardsDistributed -- an
			// ordinary ABI-encoded event with three topics -- from the same
			// receipt, and feeding that to the protobuf decoder fails the
			// whole block. The rewarding protocol writes every RewardLog with
			// Topics: nil, so the topic count is the discriminator.
			continue
		}
		logs, err := rewarding.UnmarshalRewardLog(l.Data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal receipt data into reward log")
		}
		for _, log := range logs.Logs {
			rewardLog := log
			// Classify before touching rewardInfoMap. IIP-59 added two
			// diagnostic log types that reuse the addr/amount wire slots for
			// cursor bookkeeping instead of a payout, and their addr is NOT an
			// address: EPOCH_DRAIN_OVERRUN carries "<era>:<delegates_remaining>"
			// and CURSOR_PROGRESS carries
			// "<era>:<delegate_idx>:<voter_idx>:<remaining>" with a literal "0"
			// amount. Skipping them only at the accumulation switch below would
			// be too late -- the map entry is created by the lookup, and every
			// caller iterates the map unconditionally, so a key like "12:5"
			// lands in reward_history.reward_address and block_rewards as a
			// junk row on every block of a multi-block drain.
			switch rewardLog.Type {
			case rewardingpb.RewardLog_BLOCK_REWARD,
				rewardingpb.RewardLog_EPOCH_REWARD,
				rewardingpb.RewardLog_FOUNDATION_BONUS,
				rewardingpb.RewardLog_PRIORITY_BONUS,
				rewardingpb.RewardLog_UNPRODUCTIVE_SLASH:
				// Carries a real payout to a real address; accumulate below.
			case rewardingpb.RewardLog_EPOCH_DRAIN_OVERRUN,
				rewardingpb.RewardLog_CURSOR_PROGRESS:
				// Debug rather than Warn: CURSOR_PROGRESS is emitted on every
				// block of a drain, so Warn would flood the log for hours.
				slog.L().Debug("skipping IIP-59 diagnostic reward log",
					zap.String("type", rewardLog.Type.String()),
					zap.String("addr", rewardLog.Addr),
				)
				continue
			default:
				// Deliberately not an error. The failure mode this guards
				// against is exactly what these two enum values would have
				// caused: core adds a reward log type, and the indexer halts on
				// the first block after the fork that emits it. Warn loudly,
				// keep indexing, and let the operator decide.
				slog.L().Warn("unknown reward log type, skipping",
					zap.Int32("type", int32(rewardLog.Type)),
					zap.String("addr", rewardLog.Addr),
				)
				continue
			}
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
				// Unreachable: the classification switch above already skipped
				// every type not listed here. Kept so a new payout type added
				// to that switch but forgotten here fails visibly in tests
				// rather than being silently dropped.
				return nil, errors.Errorf("reward log type %s passed classification but has no accumulator", rewardLog.Type)
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
