// Package hermes_retention is a TypeWorker plugin that periodically deletes
// stale rows from hermes_bucket_votings and hermes_aggregate_votings so they
// retain only the last N epochs (default 2185, ≈3 months at ~1 epoch/hour).
//
// The two tables are append-only (no UPDATE/DELETE in the hermes plugin) and
// every read in iotex-analyser-api filters by epoch_number, so trimming old
// rows is observable as a smaller working set with no semantic effect on the
// reward-distribution RPCs (Hermes / HermesBucket / HermesByVoter /
// HermesByDelegate) — the distributor passes startEpoch explicitly and only
// payouts that haven't yet been issued would query historical epochs.
package main

import (
	"context"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const VERSION = "1.0.0"

const (
	defaultRetentionEpochs   = uint64(2185)
	defaultTickIntervalHours = 6
)

type hermesRetentionPlugin struct {
	retentionEpochs uint64
	tickInterval    time.Duration
	stop            chan bool
	once            *sync.Once
}

func (b hermesRetentionPlugin) Name() string      { return "hermes_retention" }
func (b hermesRetentionPlugin) Type() plugin.Type { return plugin.TypeWorker }
func (b hermesRetentionPlugin) Version() string   { return VERSION }
func (b hermesRetentionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return nil
}
func (b hermesRetentionPlugin) DependentPlugins() []string { return []string{"hermes"} }

func (b *hermesRetentionPlugin) Start(ctx context.Context) error {
	b.retentionEpochs = defaultRetentionEpochs
	b.tickInterval = time.Duration(defaultTickIntervalHours) * time.Hour
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		cfg := &Config{}
		if err := yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: %s", b.Name())
		}
		if cfg.RetentionEpochs > 0 {
			b.retentionEpochs = cfg.RetentionEpochs
		}
		if cfg.TickIntervalHours > 0 {
			b.tickInterval = time.Duration(cfg.TickIntervalHours) * time.Hour
		}
	}
	slog.L().Info("hermes_retention starting",
		zap.Uint64("retentionEpochs", b.retentionEpochs),
		zap.Duration("tickInterval", b.tickInterval))

	go func() {
		// Run once on startup so retention doesn't get skipped when the
		// analyser is restarted more often than the tick interval.
		if err := b.purge(); err != nil {
			slog.L().Error("hermes_retention purge failed", zap.Error(err))
		}
		ticker := time.NewTicker(b.tickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := b.purge(); err != nil {
					slog.L().Error("hermes_retention purge failed", zap.Error(err))
				}
			case <-b.stop:
				return
			}
		}
	}()
	return nil
}

func (b *hermesRetentionPlugin) purge() error {
	var tipEpoch uint64
	if err := db.DB().Raw(
		"SELECT COALESCE(MAX(epoch_number), 0) FROM hermes_voting_meta",
	).Scan(&tipEpoch).Error; err != nil {
		return errors.Wrap(err, "failed to read tip epoch")
	}
	if tipEpoch <= b.retentionEpochs {
		return nil
	}
	cutoff := tipEpoch - b.retentionEpochs

	r1 := db.DB().Exec(
		"DELETE FROM hermes_bucket_votings WHERE epoch_number < ?", cutoff)
	if r1.Error != nil {
		return errors.Wrap(r1.Error, "failed to delete from hermes_bucket_votings")
	}
	r2 := db.DB().Exec(
		"DELETE FROM hermes_aggregate_votings WHERE epoch_number < ?", cutoff)
	if r2.Error != nil {
		return errors.Wrap(r2.Error, "failed to delete from hermes_aggregate_votings")
	}
	if r1.RowsAffected > 0 || r2.RowsAffected > 0 {
		slog.L().Info("hermes_retention purged old epochs",
			zap.Uint64("cutoff_epoch", cutoff),
			zap.Int64("bucket_deleted", r1.RowsAffected),
			zap.Int64("aggregate_deleted", r2.RowsAffected))
	}
	return nil
}

func (b hermesRetentionPlugin) Stop(ctx context.Context) error {
	b.once.Do(func() { b.stop <- true })
	return nil
}

// exported
var Plugin = hermesRetentionPlugin{
	stop: make(chan bool, 1),
	once: new(sync.Once),
}
