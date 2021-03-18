package plugins

import (
	"context"
	"database/sql"
	"math/big"
	"sync"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type runner struct {
	dao       blockdao.BlockDAO
	plugin    Plugin
	vec       prometheus.Gauge
	status    pluginStatus
	logger    *zap.Logger
	stop      chan bool
	once      *sync.Once
	isRunning *kernel.AtomicBool
	wg        sync.WaitGroup
}

func newRunner(status pluginStatus, p Plugin, dao blockdao.BlockDAO) (*runner, error) {
	r := &runner{
		dao:       dao,
		status:    status,
		plugin:    p,
		logger:    log.Logger("pluginRunner"),
		stop:      make(chan bool, 1),
		once:      new(sync.Once),
		isRunning: new(kernel.AtomicBool),
	}
	r.vec = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "iotex_analyser_metrics_" + p.Name(),

			Help: "analyser plugin metrics.",
		},
	)
	return r, nil
}

func (r *runner) UpdateStatus(status pluginStatus) {
	r.status = status
}

func (r *runner) Status() pluginStatus {
	return r.status
}

func (r *runner) nextHeight() (uint64, error) {
	height, err := r.GetHeight()
	if err != nil {
		return 0, err
	}
	nextHeight := height + 1
	return nextHeight, nil
}

func (r *runner) GetHeight() (uint64, error) {

	row := kernel.GetDB().QueryRow("SELECT height FROM index_heights WHERE name = ?", r.plugin.Name())

	var h sql.NullInt64
	if err := row.Scan(&h); err != nil && err != sql.ErrNoRows {
		return 0, errors.Wrapf(err, "failed to get index height :%s", r.plugin.Name())
	}
	if !h.Valid {
		return 0, nil
	}
	return uint64(h.Int64), nil
}

func (r *runner) Start(ctx context.Context) error {
	r.logger.Info("staring runner", zap.String("name", r.plugin.Name()))
	if err := r.plugin.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start runner")
	}
	r.isRunning.Set(true)
	var nextHeight, tipHeight uint64
	var err error

	r.wg.Add(2)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(config.Default.Iotex.PluginReportInterval)
		defer ticker.Stop()
		for {
			select {
			case <-r.stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				daoHeight, _ := r.dao.Height()
				r.logger.Info("runner report",
					zap.String("plugin", r.plugin.Name()),
					zap.Uint64("daoHeight", daoHeight),
					zap.Uint64("nextHeight", nextHeight),
				)
			}
		}
	}()
	go func() {
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
				tipHeight, err = r.dao.Height()
				if err != nil {
					r.logger.Error("failed to get blockdao height", zap.Error(err))
					continue
				}
				nextHeight, err = r.nextHeight()
				if err != nil {
					r.logger.Error("failed to get next height", zap.Error(err))
					continue
				}
				r.logger.Info("succefully to fetch plugin meta",
					zap.String("pluginName", r.plugin.Name()),
					zap.Uint64("daoHeight", tipHeight),
					zap.Uint64("nextHeight", nextHeight),
				)
				for nextHeight < tipHeight {
					if !r.isRunning.Get() {
						break
					}
					blk, err := r.dao.GetBlockByHeight(nextHeight)
					if err != nil {
						r.logger.Panic("failed to read block from dao", zap.Error(err))
					}
					receipts, err := r.dao.GetReceipts(nextHeight)
					if err != nil {
						r.logger.Panic("failed to read receipts from dao", zap.Error(err))
					}
					blk.Receipts = receipts
					actionReceipts := make(map[hash.Hash256]*action.Receipt, len(receipts))
					for _, receipt := range receipts {
						actionReceipts[receipt.ActionHash] = receipt
					}
					tlogs, err := r.dao.TransactionLogs(nextHeight)
					if err != nil {
						r.logger.Panic("failed to read transaction logs from dao", zap.Error(err))
					}
					for _, l := range tlogs.Logs {
						logs := make([]*action.TransactionLog, len(l.Transactions))
						for i, txn := range l.Transactions {
							amount, ok := new(big.Int).SetString(txn.Amount, 10)
							if !ok {
								r.logger.Panic("failed to parse", zap.Any("amount", txn.Amount))
							}
							logs[i] = &action.TransactionLog{
								Type:      txn.Type,
								Amount:    amount,
								Sender:    txn.Sender,
								Recipient: txn.Recipient,
							}
						}
						actionReceipts[hash.BytesToHash256(l.ActionHash)].AddTransactionLogs(logs...)
					}
					if err := r.plugin.PutBlock(ctx, blk); err != nil {
						r.logger.Error("failed to put data to plugin",
							zap.String("pluginName", r.plugin.Name()),
							zap.Uint64("blkHeight", blk.Height()),
							zap.Error(err),
						)
						continue
					}
					r.logger.Debug("putblock to plugin",
						zap.String("pluginName", r.plugin.Name()),
						zap.Uint64("blkHeight", blk.Height()),
					)
					r.vec.Set(float64(blk.Height()))
					nextHeight++
				}
			}
		}
	}()
	return nil
}

func (r *runner) Stop(ctx context.Context) error {
	r.once.Do(func() {
		r.logger.Info("stopping runner", zap.String("name", r.plugin.Name()))
		r.isRunning.Set(false)
		r.stop <- true
	})
	r.wg.Wait()
	if err := r.plugin.Stop(ctx); err != nil {
		return errors.Wrap(err, "failed to stop runner")
	}
	return nil
}
