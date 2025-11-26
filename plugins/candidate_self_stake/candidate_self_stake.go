package main

import (
	"context"
	"encoding/hex"
	"math"
	"slices"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
)

const VERSION = "1.0.0"

type candidateSelfStakePlugin struct {
	batchSize int
}

func (p candidateSelfStakePlugin) Name() string {
	return "candidate_self_stake"
}

func (p candidateSelfStakePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (p candidateSelfStakePlugin) DependentPlugins() []string {
	return []string{}
}

func (p *candidateSelfStakePlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(p.Name(), &models.CandidateSelfStake{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", p.Name())
	}
	p.batchSize = 1000
	return nil
}

func (p candidateSelfStakePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if err := handleBlock(ctx, blk, tx); err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, p.Name(), blk.Height())
	})
	return err
}

func (p candidateSelfStakePlugin) BatchSize() int {
	return p.batchSize
}

func (p candidateSelfStakePlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, blk := range blks {
			if err := handleBlock(ctx, blk, tx); err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, p.Name(), blks[len(blks)-1].Height())
	})
	return err
}

func (p candidateSelfStakePlugin) Stop(ctx context.Context) error {
	return nil
}

func (p candidateSelfStakePlugin) Version() string {
	return VERSION
}

// exported
var Plugin = candidateSelfStakePlugin{}

func handleBlock(ctx context.Context, blk *block.Block, tx *gorm.DB) error {
	actions := make(map[hash.Hash256]*action.SealedEnvelope, len(blk.Actions))
	for _, selp := range blk.Actions {
		actionHash, _ := selp.Hash()
		actions[actionHash] = selp
	}
	for _, receipt := range blk.Receipts {
		if receipt.Status != uint64(iotextypes.ReceiptStatus_Success) {
			continue
		}
		selp, ok := actions[receipt.ActionHash]
		if !ok {
			continue
		}
		if err := handleAction(selp.Action(), receipt.Logs(), blk.Height(), receipt.ActionHash[:], int(receipt.TxIndex), tx); err != nil {
			return err
		}
	}
	return nil
}

func handleAction(act action.Action, logs []*action.Log, blkHeight uint64, actHash []byte, txIdx int, tx *gorm.DB) error {
	switch a := act.(type) {
	case *action.CandidateRegister:
		g := kernel.Genesis()
		event, err := kernel.ParseCandidateRegisterEvent(logs, g.IsFbkMigration(blkHeight), g.IsXingu(blkHeight))
		if err != nil {
			return err
		}
		createData := models.CandidateSelfStake{
			BlockHeight: blkHeight,
			ActionHash:  hex.EncodeToString(actHash),
			CandidateID: a.OwnerAddress().String(),
			BucketID:    event.BucketID,
			TxIndex:     txIdx,
		}
		if err := upsertCandidateSelfStake(tx, &createData); err != nil {
			return err
		}
	case *action.CandidateActivate:
		event, err := kernel.ParseCandidateActivateEvent(logs)
		if err != nil {
			return err
		}
		createData := &models.CandidateSelfStake{
			BlockHeight: blkHeight,
			ActionHash:  hex.EncodeToString(actHash),
			CandidateID: event.Candidate.String(),
			BucketID:    event.BucketID,
			TxIndex:     txIdx,
		}
		if err := upsertCandidateSelfStake(tx, createData); err != nil {
			return err
		}
	case *action.CandidateEndorsement:
		if a.Op() != action.CandidateEndorsementOpRevoke {
			return nil
		}
		event, err := kernel.ParseCandidateEndorsementEvent(logs)
		if err != nil {
			return err
		}
		createData := &models.CandidateSelfStake{
			BlockHeight: blkHeight,
			ActionHash:  hex.EncodeToString(actHash),
			CandidateID: event.Candidate.String(),
			BucketID:    event.BucketID,
			TxIndex:     txIdx,
		}
		if err := upsertCandidateSelfStake(tx, createData); err != nil {
			return err
		}
	case *action.Unstake:
		bktIndex := a.BucketIndex()
		all, err := fetchCandidates(tx, blkHeight)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch latest candidate self stake records at height %d", blkHeight)
		}
		idx := slices.IndexFunc(all, func(e *models.CandidateSelfStake) bool {
			return e.BucketID == bktIndex
		})
		if idx >= 0 {
			prev := all[idx]
			createData := &models.CandidateSelfStake{
				BlockHeight: blkHeight,
				ActionHash:  hex.EncodeToString(actHash),
				CandidateID: prev.CandidateID,
				BucketID:    math.MaxUint64,
				TxIndex:     txIdx,
			}
			if err := upsertCandidateSelfStake(tx, createData); err != nil {
				return err
			}
		}
	}
	return nil
}

func fetchCandidates(tx *gorm.DB, height uint64) ([]*models.CandidateSelfStake, error) {
	// 查找出所有候选人最新的自我质押记录。即按照 candidate_id 分组，找出每组中 block_height 最大的记录。如果某个候选人在最新高度有多条记录，则都返回出来
	var results []*models.CandidateSelfStake

	err := tx.Raw(`
        SELECT css.*
        FROM candidate_self_stake css
        INNER JOIN (
            SELECT candidate_id, MAX(block_height) as max_height
            FROM candidate_self_stake
            WHERE block_height <= ?
            GROUP BY candidate_id
        ) latest ON css.candidate_id = latest.candidate_id AND css.block_height = latest.max_height
        ORDER BY css.candidate_id, css.tx_index
    `, height).Scan(&results).Error

	// only keep the record with the highest tx_index for each candidate_id
	uniqueResults := make([]*models.CandidateSelfStake, 0, len(results))
	seenCandidates := make(map[string]bool)
	for i := len(results) - 1; i >= 0; i-- {
		record := results[i]
		if !seenCandidates[record.CandidateID] {
			uniqueResults = append(uniqueResults, record)
			seenCandidates[record.CandidateID] = true
		}
	}
	// reverse the order to maintain ascending candidate_id order
	for i, j := 0, len(uniqueResults)-1; i < j; i, j = i+1, j-1 {
		uniqueResults[i], uniqueResults[j] = uniqueResults[j], uniqueResults[i]
	}

	results = uniqueResults
	return results, err
}

func upsertCandidateSelfStake(tx *gorm.DB, record *models.CandidateSelfStake) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "candidate_id"}, {Name: "action_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{"block_height", "bucket_id", "tx_index"}),
	}).Create(&record).Error
}
