package main

import (
	"context"
	"encoding/json"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/iotexproject/iotex-core/v2/pkg/unit"
	"go.uber.org/zap"
)

const VERSION = "2.2.0"

type gasTrackerPlugin struct {
	stop              chan bool
	once              *sync.Once
	dao               blockdao.BlockDAO
	calcBlockGasLimit func(height uint64) uint64
	calcBaseFee       func(height, gasUsed uint64, baseFee *big.Int) *big.Int
}

func (b gasTrackerPlugin) Name() string {
	return "gas_tracker"
}

func (b gasTrackerPlugin) Type() plugin.Type {
	return plugin.TypeWorker
}

func (b *gasTrackerPlugin) Start(ctx context.Context) error {
	dao, ok := kernel.GetBlockDAOCtx(ctx)
	if !ok {
		return errors.New("failed to get block dao")
	}
	path, ok := kernel.GetConfigCtx(ctx)
	if !ok {
		return errors.New("failed to get config path")
	}
	cfg, err := config.New(path)
	if err != nil {
		return errors.Wrap(err, "failed to load config")
	}
	b.dao = dao
	b.calcBaseFee = func(height, gasUsed uint64, baseFee *big.Int) *big.Int {
		return protocol.CalcBaseFee(cfg.Genesis.Blockchain, &protocol.TipInfo{
			Height:  height,
			GasUsed: gasUsed,
			BaseFee: baseFee,
		})
	}
	b.calcBlockGasLimit = func(height uint64) uint64 {
		return cfg.Genesis.BlockGasLimitByHeight(height)
	}

	go func() {
		if err := b.track(); err != nil {
			log.L().Error("failed to track gas", zap.Error(err))
		}

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := b.track(); err != nil {
					log.L().Error("failed to track gas", zap.Error(err))
				}
			case <-b.stop:
				return
			}
		}
	}()

	return nil
}

func (b gasTrackerPlugin) track() error {
	oracle := gasOracle{}
	recentBlocks, receipts, err := b.recentBlocks(4)
	if err != nil {
		return errors.Wrap(err, "failed to get recent blocks")
	}
	if len(recentBlocks) == 0 {
		slog.L().Warn("no recent blocks")
		return nil
	}
	tipBlk := recentBlocks[len(recentBlocks)-1]
	oracle.LastBlock = tipBlk.Height()
	oracle.GasUsedRatio = make([]float64, 0, 4)
	for _, blk := range recentBlocks {
		oracle.GasUsedRatio = append(oracle.GasUsedRatio, float64(blk.GasUsed())/float64(b.calcBlockGasLimit(blk.Height())))
	}
	oracle.SuggestBaseFee, _ = new(big.Int).Div(b.calcBaseFee(tipBlk.Height(), tipBlk.GasUsed(), tipBlk.BaseFee()), big.NewInt(unit.Qev)).Float64()
	// calculate gas price
	prices := make([]*big.Int, 0)
	for _, receipt := range receipts {
		for _, r := range receipt {
			if r.EffectiveGasPrice == nil || r.EffectiveGasPrice.Sign() == 0 {
				// ignore system action
				continue
			}
			prices = append(prices, r.EffectiveGasPrice)
		}
	}
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Cmp(prices[j]) < 0
	})
	minPrice := oracle.SuggestBaseFee + 0.1
	if len(prices) <= 1 {
		oracle.SafeGasPrice = minPrice
		oracle.ProposeGasPrice = 2 * oracle.SuggestBaseFee
		oracle.FastGasPrice = 3 * oracle.SuggestBaseFee
	} else {
		oracle.SafeGasPrice, _ = new(big.Int).Div(prices[len(prices)/10], big.NewInt(unit.Qev)).Float64()
		oracle.SafeGasPrice = max(minPrice, oracle.SafeGasPrice)
		oracle.ProposeGasPrice, _ = new(big.Int).Div(prices[len(prices)/2], big.NewInt(unit.Qev)).Float64()
		oracle.ProposeGasPrice = max(minPrice, oracle.ProposeGasPrice)
		oracle.FastGasPrice, _ = new(big.Int).Div(prices[len(prices)*9/10], big.NewInt(unit.Qev)).Float64()
		oracle.FastGasPrice = max(minPrice, oracle.FastGasPrice)
	}
	// store oracle
	raw, err := json.Marshal(oracle)
	if err != nil {
		return errors.Wrap(err, "failed to marshal oracle")
	}
	store := &db.Store{
		Key:   "iotx_gas_oracle",
		Value: string(raw),
	}
	if err := store.Save(); err != nil {
		return errors.Wrap(err, "failed to save oracle")
	}
	slog.L().Debug("store gas oracle", zap.Any("oracle", oracle))
	return nil
}

func (b gasTrackerPlugin) recentBlocks(size int) ([]*block.Block, [][]*action.Receipt, error) {
	tip, err := b.dao.Height()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get tip height")
	}
	var (
		blocks   []*block.Block
		receipts [][]*action.Receipt
	)
	for i := 1; i <= size; i++ {
		if tip < uint64(size-i)+1 {
			continue
		}
		h := tip - uint64(size-i)
		blk, err := b.dao.GetBlockByHeight(h)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to get block")
		}
		blocks = append(blocks, blk)
		receipt, err := b.dao.GetReceipts(h)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to get receipts")
		}
		receipts = append(receipts, receipt)
	}
	return blocks, receipts, nil
}

func (b gasTrackerPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return nil
}

func (b gasTrackerPlugin) Stop(ctx context.Context) error {
	b.once.Do(func() {
		b.stop <- true
	})
	return nil
}

func (b gasTrackerPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = gasTrackerPlugin{
	stop: make(chan bool, 1),
	once: new(sync.Once),
}
