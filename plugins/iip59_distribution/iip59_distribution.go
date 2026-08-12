// Package main implements the iip59_distribution plugin: it indexes the
// IIP-59 DelegateDistributed receipt logs emitted by the rewarding protocol
// when voter rewards are settled on-chain.
//
// Shape of the data it produces, because it is not one row per settlement:
// the drain is chunked across blocks, so one delegate's era settlement emits
// one log per block it was paid in. Each log fans out into one
// voter_rewards row per voter, plus one delegate_distributions
// summary row. Group on (snapshot_hash, delegate, epoch) to reassemble a
// settlement.
package main

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/distributedlog"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const VERSION = "1.0.0"

type iip59DistributionPlugin struct{}

func (b iip59DistributionPlugin) Name() string { return "iip59_distribution" }

func (b iip59DistributionPlugin) Type() plugin.Type { return plugin.TypeStandard }

// CatchUpSafe reports true: every row this plugin writes is derived from the
// logs of the block being processed, with no dependency on earlier blocks, so
// starting at the chain tip yields correct (if incomplete) history rather than
// wrong history.
func (b iip59DistributionPlugin) CatchUpSafe() bool { return true }

func (b iip59DistributionPlugin) Start(ctx context.Context) error {
	// AutoMigrate only creates tables when the plugin's index height is 0, so
	// it does nothing on an already-deployed instance. EnsureTables is what
	// actually creates these tables there.
	if err := db.AutoMigrate(b.Name(),
		&models.IIP59DelegateDistribution{},
		&models.IIP59VoterReward{},
		&models.IIP59DelegateOptIn{},
		&models.IIP59VoterDestination{},
	); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	if err := db.EnsureTables(
		&models.IIP59DelegateDistribution{},
		&models.IIP59VoterReward{},
		&models.IIP59DelegateOptIn{},
		&models.IIP59VoterDestination{},
	); err != nil {
		return errors.Wrapf(err, "failed to ensure tables for plugin %s", b.Name())
	}
	return nil
}

func (b iip59DistributionPlugin) Stop(ctx context.Context) error { return nil }

func (b iip59DistributionPlugin) Version() string { return VERSION }

func (b iip59DistributionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if err := handleBlock(blk, tx); err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
}

func (b iip59DistributionPlugin) BatchSize() int { return 0 }

func (b iip59DistributionPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	return db.DB().Transaction(func(tx *gorm.DB) error {
		for _, blk := range blks {
			if err := handleBlock(blk, tx); err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blks[len(blks)-1].Height())
	})
}

// exported
var Plugin = iip59DistributionPlugin{}

func handleBlock(blk *block.Block, tx *gorm.DB) error {
	topic0, err := distributedlog.Topic0()
	if err != nil {
		return errors.Wrap(err, "failed to derive DelegateDistributed topic")
	}

	var (
		summaries    []models.IIP59DelegateDistribution
		payouts      []models.IIP59VoterReward
		optIns       []models.IIP59DelegateOptIn
		destinations []models.IIP59VoterDestination
	)

	// Index the settings actions first. They are keyed by action hash, so they
	// need the sealed envelope for the sender, which the receipt alone does not
	// carry.
	actions := make(map[hash.Hash256]*action.SealedEnvelope, len(blk.Actions))
	for _, selp := range blk.Actions {
		h, err := selp.Hash()
		if err != nil {
			return errors.Wrap(err, "failed to hash sealed envelope")
		}
		actions[h] = selp
	}

	for _, receipt := range blk.Receipts {
		actionHash := hex.EncodeToString(receipt.ActionHash[:])

		// Only successful actions changed state. A reverted opt-in must not
		// appear as history, or the "latest row wins" read of these tables
		// returns a setting that never took effect.
		if receipt.Status == uint64(iotextypes.ReceiptStatus_Success) {
			if selp, ok := actions[receipt.ActionHash]; ok {
				sender, err := address.FromBytes(selp.SrcPubkey().Hash())
				if err != nil {
					return errors.Wrapf(err, "failed to derive sender for action %s", actionHash)
				}
				switch act := selp.Action().(type) {
				case *action.SetVoterRewardOptIn:
					cand, err := address.FromBytes(act.CandidateIdentifier())
					if err != nil {
						return errors.Wrapf(err, "failed to decode candidate identifier in action %s", actionHash)
					}
					optIns = append(optIns, models.IIP59DelegateOptIn{
						BlockHeight: blk.Height(),
						ActionHash:  actionHash,
						Sender:      sender.String(),
						Candidate:   cand.String(),
						OptIn:       act.OptIn(),
					})
				case *action.SetVoterRewardDestination:
					recipient, err := address.FromBytes(act.Recipient())
					if err != nil {
						return errors.Wrapf(err, "failed to decode recipient in action %s", actionHash)
					}
					destinations = append(destinations, models.IIP59VoterDestination{
						BlockHeight: blk.Height(),
						ActionHash:  actionHash,
						Voter:       sender.String(),
						Recipient:   recipient.String(),
					})
				}
			}
		}

		for _, l := range receipt.Logs() {
			// The address filter is load-bearing and cannot be skipped:
			// distributedlog.Unpack has no idea which contract emitted the
			// log, so without this any contract emitting the same topic
			// would be indexed as a protocol payout.
			if l.Address != address.RewardingProtocol {
				continue
			}
			if len(l.Topics) == 0 || l.Topics[0] != topic0 {
				continue
			}
			ev, err := distributedlog.Unpack(l.Topics, l.Data)
			if err != nil {
				// The address and topic already matched, so this is our event
				// with a payload we cannot read -- corruption or a format
				// change, not an unrelated log. Fail the block rather than
				// silently drop a settlement chunk.
				return errors.Wrapf(err, "failed to unpack DelegateDistributed log at height %d, action %s",
					blk.Height(), actionHash)
			}

			summaries = append(summaries, models.IIP59DelegateDistribution{
				BlockHeight:   blk.Height(),
				ActionHash:    actionHash,
				EpochNumber:   ev.Epoch,
				Delegate:      ev.Delegate.String(),
				RewardAddress: ev.RewardAddr.String(),
				SnapshotHash:  hex.EncodeToString(ev.SnapshotHash[:]),
				// Era constant, repeated every chunk. Consumers must not SUM
				// this column across a settlement; see the model doc.
				EraCommission: decimal.NewFromBigInt(ev.EraCommission, 0),
				// This chunk only; SUM across a settlement.
				ChunkVoterReward: decimal.NewFromBigInt(ev.ChunkVoterReward, 0),
				NumVoters:        uint32(len(ev.Voters)),
			})

			for i := range ev.Voters {
				payouts = append(payouts, models.IIP59VoterReward{
					BlockHeight:  blk.Height(),
					ActionHash:   actionHash,
					EpochNumber:  ev.Epoch,
					Delegate:     ev.Delegate.String(),
					SnapshotHash: hex.EncodeToString(ev.SnapshotHash[:]),
					Voter:        ev.Voters[i].String(),
					Recipient:    ev.Recipients[i].String(),
					Amount:       decimal.NewFromBigInt(ev.Amounts[i], 0),
					// Written verbatim. It is only meaningful where
					// Compounded is true -- bucket 0 is a real bucket, so
					// this column alone cannot answer "was it compounded".
					CompoundBucketID: ev.CompoundBucketIDs[i],
					Compounded:       ev.Compounded[i],
				})
			}
		}
	}

	if len(summaries) == 0 && len(optIns) == 0 && len(destinations) == 0 {
		return nil
	}

	// DoNothing on conflict keeps a replayed block idempotent. Every unique
	// index is scoped to the block, and the protocol emits each of these at
	// most once per block, so a conflict can only mean "already indexed".
	if len(summaries) > 0 {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			CreateInBatches(&summaries, 500).Error; err != nil {
			return errors.Wrap(err, "failed to insert IIP-59 delegate distributions")
		}
	}
	if len(payouts) > 0 {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			CreateInBatches(&payouts, 1000).Error; err != nil {
			return errors.Wrap(err, "failed to insert IIP-59 voter rewards")
		}
	}
	if len(optIns) > 0 {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			CreateInBatches(&optIns, 500).Error; err != nil {
			return errors.Wrap(err, "failed to insert IIP-59 delegate opt-ins")
		}
	}
	if len(destinations) > 0 {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			CreateInBatches(&destinations, 500).Error; err != nil {
			return errors.Wrap(err, "failed to insert IIP-59 voter destinations")
		}
	}

	slog.L().Debug("indexed IIP-59 data",
		zap.Uint64("height", blk.Height()),
		zap.Int("distributionLogs", len(summaries)),
		zap.Int("voterPayouts", len(payouts)),
		zap.Int("optIns", len(optIns)),
		zap.Int("destinations", len(destinations)),
	)
	return nil
}

// compile-time interface checks
var (
	_ plugin.Adapter        = iip59DistributionPlugin{}
	_ plugin.BatchAdapter   = iip59DistributionPlugin{}
	_ plugin.CatchUpAdapter = iip59DistributionPlugin{}
)
