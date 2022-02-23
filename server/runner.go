package server

import (
	"context"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	retryBlockTime = time.Microsecond * 700
)

type runner struct {
	dao       blockdao.BlockDAO
	plugin    plugin.Adapter
	vec       prometheus.Gauge
	status    pluginStatus
	logger    *zap.Logger
	isRunning *kernel.AtomicBool
	wg        sync.WaitGroup
}

func newRunner(status pluginStatus, p plugin.Adapter, dao blockdao.BlockDAO) (*runner, error) {
	r := &runner{
		dao:       dao,
		status:    status,
		plugin:    p,
		logger:    log.Logger("runner"),
		isRunning: new(kernel.AtomicBool),
	}
	r.vec = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "iotex_analyser_metrics_" + p.Name(),

			Help: "analyser plugin metrics.",
		},
	)
	prometheus.MustRegister(r.vec)
	return r, nil
}

func (r *runner) UpdateStatus(status pluginStatus) {
	r.status = status
}

func (r *runner) Status() pluginStatus {
	return r.status
}

func (r *runner) nextHeight() (uint64, error) {
	height, err := db.GetIndexHeight(r.plugin.Name())
	if err != nil {
		return 0, err
	}
	nextHeight := height + 1
	return nextHeight, nil
}

func (r *runner) Start(ctx context.Context) error {
	r.logger.Info("starting runner", zap.String("name", r.plugin.Name()))
	switch r.plugin.Type() {
	case plugin.TypeWorker:
		if err := r.plugin.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start runner")
		}
		r.isRunning.Set(true)
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
		}()
		return nil
	case plugin.TypeStandard:
	default:
	}
	if config.Default.Iotex.CatchUpMode {
		if err := db.DB().Transaction(func(tx *gorm.DB) error {
			var daoHeight uint64
			for i := 0; i < 5; i++ {
				daoHeight, _ = r.dao.Height()
				if daoHeight == 0 {
					r.logger.Warn("retrying fetch dao height")
					time.Sleep(time.Second)
					continue
				}
				r.logger.Info("daoheight", zap.Uint64("daoheight", daoHeight), zap.String("name", r.plugin.Name()))
				break
			}
			if daoHeight == 0 {
				return errors.New("failed to fetch dao height")
			}
			if config.Default.Iotex.CatchUpStartHeight > 0 {
				daoHeight = config.Default.Iotex.CatchUpStartHeight - 1
			}
			return db.UpdateIndexHeightByTx(tx, r.plugin.Name(), daoHeight)
		}); err != nil {
			return err
		}
	}
	if err := r.plugin.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start runner")
	}
	if config.Default.Iotex.CrawlMode {
		for _, nextHeight := range config.Default.Iotex.CrawlHeight {
			blk, err := kernel.GetBlockByHeightFromChain(nextHeight)
			if err != nil {
				r.logger.Error("failed to read block from chain")
				continue
			}
			if err := r.plugin.PutBlock(ctx, blk); err != nil {
				r.logger.Error("failed to put data to plugin",
					zap.String("pluginName", r.plugin.Name()),
					zap.Uint64("blkHeight", blk.Height()),
					zap.Error(err),
				)
			}
			r.logger.Debug("putblock to plugin",
				zap.String("pluginName", r.plugin.Name()),
				zap.Uint64("blkHeight", blk.Height()),
			)
		}
		return nil
	}
	r.isRunning.Set(true)
	r.wg.Add(1)
	go func() {
		var nextHeight, tipHeight uint64
		var err error
		var blk *block.Block
		defer r.wg.Done()
		for {
			if !r.isRunning.Get() {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
				//prevent dead loop
				time.Sleep(3 * time.Second)
				tipHeight, err = r.getTipHeight()
				if err != nil {
					r.logger.Error("failed to get blockdao height, retrying...", zap.Error(err))
					continue
				}
				nextHeight, err = r.nextHeight()
				if err != nil {
					r.logger.Error("failed to get next height, retrying...", zap.Error(err))
					continue
				}
				r.logger.Debug("succefully to fetch plugin meta",
					zap.String("pluginName", r.plugin.Name()),
					zap.Uint64("daoHeight", tipHeight),
					zap.Uint64("nextHeight", nextHeight),
				)
				for nextHeight <= tipHeight {
					if !r.isRunning.Get() {
						break
					}

					if !config.Default.Iotex.CatchUpMode {
						blk, err = kernel.GetBlockByHeightFromBlockDAO(nextHeight, r.dao)
					} else {
						blk, err = kernel.GetBlockByHeightFromChain(nextHeight)
					}
					if err != nil {
						r.logger.Error("failed to read block from dao",
							zap.Error(err),
							zap.String("pluginName", r.plugin.Name()),
							zap.Uint64("height", nextHeight),
							zap.Uint64("tipHeight", tipHeight),
						)
						break
					}
					if err := r.plugin.PutBlock(ctx, blk); err != nil {
						r.logger.Error("failed to put data to plugin, it will be retry in next time",
							zap.String("pluginName", r.plugin.Name()),
							zap.Uint64("height", blk.Height()),
							zap.Error(err),
						)
						break
					}
					r.logger.Debug("putblock to plugin",
						zap.String("pluginName", r.plugin.Name()),
						zap.Uint64("height", blk.Height()),
					)
					r.vec.Set(float64(blk.Height()))
					nextHeight++
				}
			}
		}
	}()
	return nil
}

//checking dependent plugin
func (r *runner) getTipHeight() (uint64, error) {
	if dep, ok := r.plugin.(plugin.DependentAdapter); ok {
		return db.GetIndexHeight(dep.DependentPlugin())
	}
	return r.dao.Height()
}

func (r *runner) Stop(ctx context.Context) error {
	r.logger.Info("stopping runner", zap.String("name", r.plugin.Name()))
	r.isRunning.Set(false)
	r.wg.Wait()
	if err := r.plugin.Stop(ctx); err != nil {
		return errors.Wrap(err, "failed to stop runner")
	}
	return nil
}
