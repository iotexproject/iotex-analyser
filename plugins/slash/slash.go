package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"slices"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert/yaml"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const VERSION = "2.1.2"

type slashPlugin struct {
}

func (b slashPlugin) Name() string {
	return "slash"
}

func (b slashPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b slashPlugin) DependentPlugins() []string {
	return []string{"candidate", "candidate_self_stake"}
}

func (b slashPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.Slash{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	height, err := db.GetIndexHeight(b.Name())
	if err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	startHeight := uint64(0)
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		cfg := &Config{}
		if err = yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			startHeight = cfg.StartHeight
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}
	if height < startHeight {
		return db.UpdateIndexHeight(b.Name(), startHeight-1)
	}
	return nil
}

func (b slashPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		var epochRewardHash []byte
		// log action index
		for i := len(blk.Actions) - 1; i >= 0; i-- {
			selp := blk.Actions[i]
			if grantReward, ok := selp.Action().(*action.GrantReward); ok && grantReward.RewardType() == action.EpochReward {
				actHash, _ := selp.Hash()
				epochRewardHash = actHash[:]
				break
			}
		}
		if len(epochRewardHash) == 0 {
			// no epoch reward in this block
			return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
		}
		// log receipt index
		idx := slices.IndexFunc(blk.Receipts, func(e *action.Receipt) bool {
			return bytes.Equal(e.ActionHash[:], epochRewardHash)
		})
		if idx == -1 {
			return errors.Errorf("cannot find receipt for epoch reward action %x", epochRewardHash)
		}
		receipt := blk.Receipts[idx]
		// Parse receipt of grant reward
		rewardInfoMap, err := kernel.RewardInfoFromReceipt(receipt)
		if err != nil {
			return err
		}

		for addr, slash := range rewardInfoMap {
			if slash.UnproductiveSlash == nil || slash.UnproductiveSlash.Sign() == 0 {
				continue
			}
			cand := &models.Candidate{}
			if err := cand.FetchByOperatorAddressWithHeight(addr, blkHeight, tx); err != nil {
				return err
			}
			css := &models.CandidateSelfStake{}
			if err := css.FetchByCandidateIDWithHeight(cand.CandidateID, blkHeight, tx); err != nil {
				return err
			}
			m := models.Slash{
				BlockHeight:     blkHeight,
				ActionHash:      hex.EncodeToString(receipt.ActionHash[:]),
				OperatorAddress: addr,
				Amount:          decimal.NewFromBigInt(slash.UnproductiveSlash, 0),
				CandidateID:     cand.CandidateID,
				BucketID:        css.BucketID,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "block_height"}, {Name: "operator_address"}},
				DoUpdates: clause.AssignmentColumns([]string{"action_hash", "amount", "candidate_id", "bucket_id"}),
			}).Create(&m).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err

}

func (b slashPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b slashPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = slashPlugin{}
