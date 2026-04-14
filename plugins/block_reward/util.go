package main

import (
	"encoding/hex"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/millken/gocache"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func getCandidateNameByAddress(addr string) (string, error) {
	name, err := gocache.Memoize("000"+addr, func() (interface{}, error) {
		var cand models.Candidate
		if err := db.DB().Model(cand).Where("reward_address=?", addr).Order("id desc").Take(&cand).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
		return cand.Name, nil
	}, time.Minute*10)
	return name.(string), err
}

func collectRewardRows(blkHeight uint64, epochNum uint64, receipt *action.Receipt, rewardInfoMap map[string]*kernel.RewardInfo) ([]models.BlockReward, error) {
	if len(rewardInfoMap) == 0 {
		return nil, nil
	}

	var rows []models.BlockReward
	for addr, reward := range rewardInfoMap {
		candName, err := getCandidateNameByAddress(addr)
		if err != nil {
			return nil, err
		}
		rows = append(rows, models.BlockReward{
			BlockHeight:     blkHeight,
			EpochNumber:     epochNum,
			RewardAddress:   addr,
			ActionHash:      hex.EncodeToString(receipt.ActionHash[:]),
			CandidateName:   candName,
			BlockReward:     decimal.NewFromBigInt(reward.BlockReward, 0),
			EpochReward:     decimal.NewFromBigInt(reward.EpochReward, 0),
			FoundationBonus: decimal.NewFromBigInt(reward.FoundationBonus, 0),
			PriorityBonus:   decimal.NewFromBigInt(reward.PriorityBonus, 0),
		})
	}
	return rows, nil
}
