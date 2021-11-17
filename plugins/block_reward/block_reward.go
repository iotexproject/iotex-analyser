package main

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/action/protocol/rewarding/rewardingpb"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const VERSION = "2.0.0"

type blockRewardPlugin struct {
}

type RewardInfo struct {
	BlockReward     *big.Int
	EpochReward     *big.Int
	FoundationBonus *big.Int
}

func (b blockRewardPlugin) Name() string {
	return "block_reward"
}

func (b blockRewardPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockRewardPlugin) Start(ctx context.Context) error {
	if err := db.DB().AutoMigrate(&models.BlockReward{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b blockRewardPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)

	grantRewardActs := make(map[hash.Hash256]bool)
	// log action index
	for _, selp := range blk.Actions {
		if _, ok := selp.Action().(*action.GrantReward); ok {
			actHash := selp.Hash()
			grantRewardActs[actHash] = true
		}
	}
	// log receipt index
	for _, receipt := range blk.Receipts {
		if _, ok := grantRewardActs[receipt.ActionHash]; !ok {
			continue
		}
		// Parse receipt of grant reward
		rewardInfoMap := make(map[string]*RewardInfo)
		for _, l := range receipt.Logs() {
			rewardLog := &rewardingpb.RewardLog{}
			if err := proto.Unmarshal(l.Data, rewardLog); err != nil {
				return errors.Wrap(err, "failed to unmarshal receipt data into reward log")
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
				return errors.New("failed to convert reward amount from string to big int")
			}
			switch rewardLog.Type {
			case rewardingpb.RewardLog_BLOCK_REWARD:
				rewards.BlockReward.Add(rewards.BlockReward, amount)
			case rewardingpb.RewardLog_EPOCH_REWARD:
				rewards.EpochReward.Add(rewards.EpochReward, amount)
			case rewardingpb.RewardLog_FOUNDATION_BONUS:
				rewards.FoundationBonus.Add(rewards.FoundationBonus, amount)
			default:
				return errors.New("Unknown type of reward")
			}
		}

		if len(rewardInfoMap) == 0 {
			continue
		}
		err := db.DB().Transaction(func(tx *gorm.DB) error {

			for addr, reward := range rewardInfoMap {
				var cand models.Candidate
				if err := tx.Model(cand).Where("reward_address=?", addr).Order("id desc").Take(&cand).Error; err != nil {
					log.L().Warn("can not fetch candidate name", zap.String("reward_address", addr))
				}
				m := models.BlockReward{
					BlockHeight:     blkHeight,
					EpochNumber:     epochNum,
					RewardAddress:   addr,
					ActionHash:      hex.EncodeToString(receipt.ActionHash[:]),
					CandidateName:   cand.Name,
					BlockReward:     decimal.NewFromBigInt(reward.BlockReward, 0),
					EpochReward:     decimal.NewFromBigInt(reward.EpochReward, 0),
					FoundationBonus: decimal.NewFromBigInt(reward.FoundationBonus, 0),
				}
				if err := tx.Create(&m).Error; err != nil {
					return err
				}
			}
			return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
		})
		return err
	}

	return nil
}

func (b blockRewardPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockRewardPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockRewardPlugin{}
