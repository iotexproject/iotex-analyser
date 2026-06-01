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

const VERSION = "2.1.4"

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
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if err := handleBlock(ctx, blk, tx); err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err

}

func (b slashPlugin) BatchSize() int {
	return 0
}

func (b slashPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, blk := range blks {
			if err := handleBlock(ctx, blk, tx); err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blks[len(blks)-1].Height())
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

func handleBlock(ctx context.Context, blk *block.Block, tx *gorm.DB) error {
	blkHeight := blk.Height()
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
		return nil
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
		// The slash RewardLog's `Addr` follows the iotex-core rewarding
		// protocol version active at this block. Older blocks used the
		// candidate's operator address (SlashCandidateByOperator). Blocks
		// after the slash-by-owner / slash-by-ID upgrade carry the
		// candidate's owner address (which is also the candidate_id).
		// Look up by either so both eras work.
		cand := &models.Candidate{}
		if err := cand.FetchByOwnerOrOperatorWithHeight(addr, blkHeight, tx); err != nil {
			return err
		}
		// A candidate may have been registered without ever issuing a
		// CandidateSelfStake action (e.g. registered with zero self-stake).
		// In that case BucketID stays at its zero value — it is informational
		// in the slash record, not a foreign key.
		css := &models.CandidateSelfStake{}
		if err := css.FetchByCandidateIDWithHeight(cand.CandidateID, blkHeight, tx); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
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
	return nil
}
