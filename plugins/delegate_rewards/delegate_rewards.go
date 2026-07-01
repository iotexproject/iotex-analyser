// Package delegate_rewards is a TypeWorker plugin that maintains the
// per-candidate cumulative rewards rollup table `delegate_rewards`. It
// replaces the legacy iotex-analytics sync_delegate_rewards() proc that was
// dropped during the migration to iotex-analyser, and unbreaks iotex-kit's
// getDelegateRewards() (hub "All Time" rewards view).
//
// Each tick re-aggregates hermes_account_rewards (block/epoch/foundation)
// and staking_actions burn-drop DepositToStake into a single INSERT ... ON
// CONFLICT UPDATE keyed by candidate address. Because the query is
// idempotent, missed ticks are self-healing; a daily cadence is enough for
// the "All Time" use case.
package main

import (
	"context"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const VERSION = "1.0.0"

// Give hermes / staking_actions time to catch up on cold start before the
// first sync tick fires.
const warmupDelay = 5 * time.Minute
const tickInterval = 24 * time.Hour

type delegateRewardsPlugin struct {
	stop chan bool
	once *sync.Once
}

func (b delegateRewardsPlugin) Name() string      { return "delegate_rewards" }
func (b delegateRewardsPlugin) Type() plugin.Type { return plugin.TypeWorker }
func (b delegateRewardsPlugin) Version() string   { return VERSION }
func (b delegateRewardsPlugin) DependentPlugins() []string {
	return []string{"hermes", "staking_actions"}
}

func (b delegateRewardsPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.DelegateRewards{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	go func() {
		select {
		case <-b.stop:
			return
		case <-time.After(warmupDelay):
		}
		if err := syncDelegateRewards(); err != nil {
			log.L().Warn("delegate_rewards initial sync failed", zap.Error(err))
		}
		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-b.stop:
				return
			case <-ticker.C:
				if err := syncDelegateRewards(); err != nil {
					log.L().Warn("delegate_rewards sync failed", zap.Error(err))
				}
			}
		}
	}()
	return nil
}

func (b delegateRewardsPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return nil
}

func (b delegateRewardsPlugin) Stop(ctx context.Context) error {
	b.once.Do(func() {
		b.stop <- true
	})
	return nil
}

// CatchUpSafe: the plugin re-derives its state from a snapshot query at each
// tick, so starting the analyser in catch-up mode does not corrupt it.
func (b delegateRewardsPlugin) CatchUpSafe() bool { return true }

// exported
var Plugin = delegateRewardsPlugin{
	once: new(sync.Once),
	stop: make(chan bool, 1),
}
