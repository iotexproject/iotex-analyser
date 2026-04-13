package server

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	retryBlockTime = time.Microsecond * 700
)

var (
	activeRunners = make(map[string]*runner)
	activeLock    = new(sync.RWMutex)
)

func setRunner(name string, r *runner) {
	activeLock.Lock()
	activeRunners[name] = r
	activeLock.Unlock()
}

func getRunner(name string) (*runner, bool) {
	activeLock.RLock()
	r, ok := activeRunners[name]
	activeLock.RUnlock()
	return r, ok
}

func setRunners(r map[string]*runner) {
	activeLock.Lock()
	activeRunners = r
	activeLock.Unlock()
}

func getRunners() map[string]*runner {
	activeLock.RLock()
	r := activeRunners
	activeLock.RUnlock()
	return r
}

func GetRunnerStats() RunnerStats {
	var stats RunnerStats
	stats.Server = ServerStat{
		DaoHeight: atomic.LoadUint64(&_daoHeight),
		TipHeight: atomic.LoadUint64(&_tipHeight),
	}
	for name, r := range getRunners() {
		stats.Runners = append(stats.Runners, RunnerStat{
			Name:         name,
			PluginType:   r.plugin.Type(),
			PluginStatus: r.Status(),
			Error:        r.Error(),
		})
	}
	return stats
}

type runner struct {
	dao          kernel.BatchBlockDao
	plugin       plugin.Adapter
	status       pluginStatus
	logger       *zap.Logger
	isRunning    *kernel.AtomicBool
	wg           *sync.WaitGroup
	err          error
	mu           sync.RWMutex
	batchSizeMgr *batchSizeManager // 批次大小管理器

	// stats tracking
	startNano       atomic.Int64
	startHeight     atomic.Uint64
	blocksProcessed atomic.Uint64
	txsProcessed    atomic.Uint64
}

func newRunner(status pluginStatus, p plugin.Adapter, dao kernel.BatchBlockDao) (*runner, error) {
	r := &runner{
		dao:       dao,
		status:    status,
		plugin:    p,
		logger:    log.Logger("runner"),
		isRunning: new(kernel.AtomicBool),
		wg:        &sync.WaitGroup{},
	}
	return r, nil
}

func (r *runner) UpdateStatus(status pluginStatus) {
	r.mu.Lock()
	r.status = status
	r.mu.Unlock()
}

func (r *runner) Status() pluginStatus {
	r.mu.RLock()
	status := r.status
	r.mu.RUnlock()
	return status
}

func (r *runner) UpdateError(err error) {
	r.mu.Lock()
	r.err = err
	r.mu.Unlock()
}

func (r *runner) Error() error {
	r.mu.RLock()
	err := r.err
	r.mu.RUnlock()
	return err
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
			timeStart := time.Now()
			blk, err := kernel.GetBlockByHeightFromChain(ctx, nextHeight)
			if err != nil {
				r.logger.Error("failed to read block from chain", zap.Error(err))
				continue
			}
			if err := r.plugin.PutBlock(ctx, blk); err != nil {
				r.logger.Error("failed to put data to plugin",
					zap.String("pluginName", r.plugin.Name()),
					zap.Uint64("blkHeight", blk.Height()),
					zap.Error(err),
				)
			}
			pluginProcessingSecondsPerBlockMetrics.WithLabelValues(r.plugin.Name()).Observe(time.Since(timeStart).Seconds())
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

		// 初始化批次大小管理器
		initialBatchSize := uint64(config.Default.BlockDB.BatchSize)
		if p, ok := r.plugin.(plugin.BatchAdapter); ok {
			if p.BatchSize() > 0 {
				initialBatchSize = uint64(p.BatchSize())
			}
		}
		r.batchSizeMgr = newBatchSizeManager(initialBatchSize, r.plugin.Name(), r.logger)

		statsInitialized := false
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
				if !statsInitialized {
					r.startHeight.Store(nextHeight - 1)
					r.startNano.Store(time.Now().UnixNano())
					statsInitialized = true
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

					blks := make([]*block.Block, 0, 5000)

					timeStart := time.Now()
					if !config.Default.Iotex.CatchUpMode {
						blk, err = kernel.GetBlockByHeightFromBlockDAO(nextHeight, r.dao)

						if _, ok := r.plugin.(plugin.BatchAdapter); ok {
							count := tipHeight - nextHeight + 1
							currentBatchSize := r.batchSizeMgr.getCurrent()
							if currentBatchSize > 0 && currentBatchSize < count {
								count = currentBatchSize
							}
							blks, err = r.fetchBlocks(nextHeight, count)
							if err != nil {
								r.logger.Error("failed to batch read blocks from dao",
									zap.Error(err),
									zap.String("pluginName", r.plugin.Name()),
									zap.Uint64("height", nextHeight),
									zap.Uint64("tipHeight", tipHeight),
								)
							} else {
								// 成功获取，尝试增加批次大小
								r.batchSizeMgr.onSuccess()
								// limit blocks by total tx count to prevent OOM on high-tx blocks
								// Use count * 500 as the tx limit (e.g., 1000 blocks * 500 = 500K txs max).
								// Previously maxTxs was set to count (block count), which conflated
								// block count with tx count — at high-tx heights (~350 txs/block),
								// this truncated batches to ~3 blocks, wasting 99.7% of fetched data.
								maxTxs := count * 500
								txCount := uint64(0)
								for i, blk := range blks {
									txCount += uint64(len(blk.Actions))
									if txCount > maxTxs {
										blks = blks[:i+1]
										break
									}
								}
							}
						}
					} else {
						blk, err = kernel.GetBlockByHeightFromChain(ctx, nextHeight)
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

					putBlocks := func() (exit bool) {
						defer func() {
							if rc := recover(); rc != nil {
								r.UpdateStatus(PluginStatusPutError)
								r.UpdateError(errors.Errorf("panic when putting block to plugin: %v", rc))
								r.logger.Error("panic when putting block to plugin",
									zap.String("pluginName", r.plugin.Name()),
									zap.Uint64("height", nextHeight),
									zap.Error(r.Error()),
								)
								exit = true
							}
						}()

						if batchPlugin, ok := r.plugin.(plugin.BatchAdapter); ok {
							if err := batchPlugin.PutBlocks(ctx, blks); err != nil {
								r.UpdateStatus(PluginStatusPutError)
								r.UpdateError(err)
								r.logger.Error("failed to put blocks to plugin, retrying...",
									zap.String("pluginName", r.plugin.Name()),
									zap.Uint64("height", nextHeight),
									zap.Int("batchSize", len(blks)),
									zap.Error(err),
								)
								return true
							}
							r.UpdateStatus(PluginStatusPutOK)
							r.UpdateError(nil)
							r.logger.Debug("putBlocks to plugin",
								zap.String("pluginName", r.plugin.Name()),
								zap.Uint64("height", nextHeight),
								zap.Int("batchSize", len(blks)),
							)
							serverMetrics.WithLabelValues("plugin", r.plugin.Name()).Set(float64(nextHeight))
							if len(blks) > 0 {
								pluginProcessingSecondsPerBlockMetrics.WithLabelValues(r.plugin.Name()).Observe(time.Since(timeStart).Seconds() / float64(len(blks)))
								var txCount uint64
								for _, b := range blks {
									txCount += uint64(len(b.Actions))
								}
								r.blocksProcessed.Add(uint64(len(blks)))
								r.txsProcessed.Add(txCount)
							}

							nextHeight += uint64(len(blks))
							blks = blks[:0]
							return false
						}

						if err := r.plugin.PutBlock(ctx, blk); err != nil {
							r.UpdateStatus(PluginStatusPutError)
							r.UpdateError(err)
							r.logger.Error("failed to put data to plugin, retrying...",
								zap.String("pluginName", r.plugin.Name()),
								zap.Uint64("height", blk.Height()),
								zap.Error(err),
							)
							return true
						}
						r.UpdateStatus(PluginStatusPutOK)
						r.UpdateError(nil)
						r.logger.Debug("putblock to plugin",
							zap.String("pluginName", r.plugin.Name()),
							zap.Uint64("height", blk.Height()),
						)
						serverMetrics.WithLabelValues("plugin", r.plugin.Name()).Set(float64(blk.Height()))
						pluginProcessingSecondsPerBlockMetrics.WithLabelValues(r.plugin.Name()).Observe(time.Since(timeStart).Seconds())
						r.blocksProcessed.Add(1)
						r.txsProcessed.Add(uint64(len(blk.Actions)))
						nextHeight++
						return false
					}

					if putBlocks() {
						break
					}
				}
			}
		}
	}()
	return nil
}

// checking dependent plugin
func (r *runner) getTipHeight() (uint64, error) {
	depHeight, err := r.dao.Height()
	if err != nil {
		return depHeight, err
	}
	if dep, ok := r.plugin.(plugin.DependentAdapter); ok {
		for _, pluginName := range dep.DependentPlugins() {
			pluginHeight, err := db.GetIndexHeight(pluginName)
			if err != nil {
				return pluginHeight, err
			}
			if depHeight > pluginHeight {
				depHeight = pluginHeight
			}
		}
	}
	return depHeight, nil
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

func (r *runner) logStats(logger *zap.Logger) {
	startNano := r.startNano.Load()
	if startNano == 0 {
		return
	}
	elapsed := float64(time.Now().UnixNano()-startNano) / float64(time.Second)
	if elapsed < 1 {
		return
	}
	blocks := r.blocksProcessed.Load()
	txs := r.txsProcessed.Load()
	blocksPerSec := float64(blocks) / elapsed
	txsPerSec := float64(txs) / elapsed

	currentHeight := r.startHeight.Load() + blocks

	logger.Info("plugin processing stats",
		zap.String("plugin", r.plugin.Name()),
		zap.Uint64("currentHeight", currentHeight),
		zap.Uint64("blocksProcessed", blocks),
		zap.Uint64("txsProcessed", txs),
		zap.Float64("blocks/sec", blocksPerSec),
		zap.Float64("txs/sec", txsPerSec),
	)
}

func (r *runner) fetchBlocks(start, count uint64) ([]*block.Block, error) {
	if count == 0 {
		return []*block.Block{}, nil
	}
	blks, err := r.dao.BatchGetBlocks(start, count)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.ResourceExhausted {
				// 减小批次大小并重试
				newCount := r.batchSizeMgr.onFailure(count)
				r.logger.Warn("reducing batch size and retrying fetch blocks",
					zap.String("pluginName", r.plugin.Name()),
					zap.Uint64("start", start),
					zap.Uint64("oldCount", count),
					zap.Uint64("newCount", newCount),
					zap.Error(err),
				)
				return r.fetchBlocks(start, newCount)
			}
		}
		return nil, errors.Wrap(err, "failed to fetch blocks from dao")
	}
	return blks, nil
}
