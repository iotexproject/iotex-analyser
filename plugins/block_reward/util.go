package main

import (
	"encoding/hex"
	"math/big"
	"time"

	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/rewardingpb"
	"github.com/millken/gocache"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

func getCandidateNameByAddress(tx *gorm.DB, addr string) (string, error) {
	name, err := gocache.Memoize("000"+addr, func() (interface{}, error) {
		var cand models.Candidate
		if err := tx.Model(cand).Where("reward_address=?", addr).Order("id desc").Take(&cand).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
		return cand.Name, nil
	}, time.Minute*10)
	return name.(string), err
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

func handleRewardInfoMap(tx *gorm.DB, blkHeight uint64, epochNum uint64, receipt *action.Receipt, rewardInfoMap map[string]*RewardInfo) error {
	if len(rewardInfoMap) == 0 {
		return nil
	}

	for addr, reward := range rewardInfoMap {
		candName, err := getCandidateNameByAddress(tx, addr)
		if err != nil {
			return err
		}
		m := models.BlockReward{
			BlockHeight:     blkHeight,
			EpochNumber:     epochNum,
			RewardAddress:   addr,
			ActionHash:      hex.EncodeToString(receipt.ActionHash[:]),
			CandidateName:   candName,
			BlockReward:     decimal.NewFromBigInt(reward.BlockReward, 0),
			EpochReward:     decimal.NewFromBigInt(reward.EpochReward, 0),
			FoundationBonus: decimal.NewFromBigInt(reward.FoundationBonus, 0),
		}
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}
