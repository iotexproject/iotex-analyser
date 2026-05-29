package main

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const VERSION = "1.0.0"

const (
	// storeKey is the db.Store key holding the cumulative burned (base gas fee) state.
	storeKey = "total_burned_gasfee"
	// gasFeeType is the block_receipt_transactions.type value for base gas fee.
	gasFeeType = "gasFee"
	// chunkSize bounds how many block heights a single SUM query spans.
	chunkSize = uint64(1000000)
	// tickInterval is how often the worker catches up to the chain tip.
	tickInterval = 30 * time.Second
)

// burnedState is the JSON payload persisted in the store KV.
type burnedState struct {
	BlockHeight uint64 `json:"block_height"` // watermark: highest height summed so far
	Amount      string `json:"amount"`       // cumulative base gas fee, rau (decimal string)
}

type gasBurnedPlugin struct {
	stop chan bool
	once *sync.Once
}

func (b gasBurnedPlugin) Name() string {
	return "gas_burned"
}

func (b gasBurnedPlugin) Type() plugin.Type {
	return plugin.TypeWorker
}

func (b *gasBurnedPlugin) Start(ctx context.Context) error {
	go func() {
		if err := b.track(); err != nil {
			log.L().Error("failed to track burned gas fee", zap.Error(err))
		}

		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := b.track(); err != nil {
					log.L().Error("failed to track burned gas fee", zap.Error(err))
				}
			case <-b.stop:
				return
			}
		}
	}()
	return nil
}

// track incrementally sums new gasFee transaction logs from
// block_receipt_transactions and advances the persisted cumulative total.
func (b gasBurnedPlugin) track() error {
	last, amount, err := loadState()
	if err != nil {
		return errors.Wrap(err, "failed to load burned state")
	}

	var maxRow struct{ Height uint64 }
	if err := db.DB().Raw(
		"SELECT COALESCE(MAX(block_height), 0) AS height FROM block_receipt_transactions",
	).Scan(&maxRow).Error; err != nil {
		return errors.Wrap(err, "failed to read max block height")
	}
	target := maxRow.Height
	if target <= last {
		return nil
	}

	for last < target {
		to := last + chunkSize
		if to > target {
			to = target
		}
		var sumRow struct{ Sum decimal.Decimal }
		if err := db.DB().Raw(
			"SELECT COALESCE(SUM(amount), 0) AS sum FROM block_receipt_transactions WHERE type = ? AND block_height > ? AND block_height <= ?",
			gasFeeType, last, to,
		).Scan(&sumRow).Error; err != nil {
			return errors.Wrap(err, "failed to sum gas fee window")
		}
		amount = amount.Add(sumRow.Sum)
		last = to
		if err := saveState(last, amount); err != nil {
			return errors.Wrap(err, "failed to save burned state")
		}
	}
	return nil
}

func loadState() (uint64, decimal.Decimal, error) {
	s := &db.Store{Key: storeKey}
	if err := s.Get(); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, decimal.Zero, nil
		}
		return 0, decimal.Zero, err
	}
	var st burnedState
	if err := json.Unmarshal([]byte(s.Value), &st); err != nil {
		return 0, decimal.Zero, err
	}
	amount, err := decimal.NewFromString(st.Amount)
	if err != nil {
		return 0, decimal.Zero, err
	}
	return st.BlockHeight, amount, nil
}

func saveState(last uint64, amount decimal.Decimal) error {
	raw, err := json.Marshal(burnedState{BlockHeight: last, Amount: amount.String()})
	if err != nil {
		return err
	}
	s := &db.Store{Key: storeKey, Value: string(raw)}
	return s.Save()
}

func (b gasBurnedPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return nil
}

func (b gasBurnedPlugin) Stop(ctx context.Context) error {
	b.once.Do(func() {
		b.stop <- true
	})
	return nil
}

func (b gasBurnedPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = gasBurnedPlugin{
	stop: make(chan bool, 1),
	once: new(sync.Once),
}
