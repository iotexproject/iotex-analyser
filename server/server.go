package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/pprof"
	"net/rpc"
	"net/url"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/go-pkgs/util/httputil"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/v2/blockchain/filedao"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/iotexproject/iotex-core/v2/server/itx"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	OpDurationMtc = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "plugin_op_metrics",
		Help: "plugin op metrics.",
	}, []string{"plugin", "op"})

	serverMetrics = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "iotex_analyser_status",
			Help: "iotex analyser plugin status",
		},
		[]string{"type", "name"},
	)
	pluginProcessingSecondsPerBlockMetrics = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "iotex_analyser_plugin_processing_seconds_per_block",
			Help:       "iotex analyser plugin processing seconds per block",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"name"},
	)
	_tipHeight, _daoHeight uint64
)

type Server struct {
	m         sync.RWMutex
	dao       kernel.BatchBlockDao
	service   *Service
	logger    *zap.Logger
	isRunning *kernel.AtomicBool
}

func New() *Server {
	s := &Server{
		logger:    log.Logger("server"),
		isRunning: new(kernel.AtomicBool),
	}
	return s
}

// Start start the server
func (srv *Server) Start(ctx context.Context) error {
	prometheus.MustRegister(OpDurationMtc)

	prometheus.MustRegister(serverMetrics)
	prometheus.MustRegister(pluginProcessingSecondsPerBlockMetrics)

	_, err := db.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect DB")
	}
	// if err := kernel.GetDB().Ping(); err != nil {
	// 	return errors.Wrap(err, "failed to ping DB")
	// }
	srv.isRunning.Set(true)

	if err := srv.startDaoService(ctx); err != nil {
		return errors.Wrap(err, "failed to start blockdao service")
	}

	if config.Default.Server.Http != "" {
		go func() {
			if err := srv.startHTTPService(); err != nil {
				srv.logger.Fatal("failed to start http service", zap.Error(err))
			}
		}()
	}

	srv.startDebugService(ctx)
	srv.logger.Info("start RPC service")
	if err := srv.startRPCService(ctx); err != nil {
		return errors.Wrap(err, "failed to start RPC service")
	}
	return nil
}

func (srv *Server) startDebugService(ctx context.Context) {
	var adminserv http.Server
	if config.Default.Server.HTTPAdminPort > 0 {
		mux := http.NewServeMux()
		log.RegisterLevelConfigMux(mux)
		mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

		port := fmt.Sprintf(":%d", config.Default.Server.HTTPAdminPort)
		adminserv = httputil.Server(port, mux)
		// defer func() {
		// 	if err := adminserv.Shutdown(ctx); err != nil {
		// 		log.L().Error("Error when serving metrics data.", zap.Error(err))
		// 	}
		// }()
		go func() {
			runtime.SetMutexProfileFraction(1)
			runtime.SetBlockProfileRate(1)
			ln, err := httputil.LimitListener(adminserv.Addr)
			if err != nil {
				log.L().Error("Error when listen to profiling port.", zap.Error(err))
				return
			}
			if err := adminserv.Serve(ln); err != nil {
				log.L().Error("Error when serving performance profiling data.", zap.Error(err))
			}
		}()
	}
}

func (srv *Server) Stop(ctx context.Context) error {
	srv.m.RLock()
	defer srv.m.RUnlock()
	sockAddr := config.Default.Server.Addr
	if _, err := os.Stat(sockAddr); err == nil {
		os.Remove(sockAddr)
	}
	srv.isRunning.Set(false)
	if err := srv.service.Stop(ctx); err != nil {
		return err
	}
	return srv.dao.Stop(ctx)
}

func (srv *Server) startRebuildBlockDaoWorker(ctx context.Context) error {
	var daoHeight uint64
	chainClient := kernel.ChainClient()
	res, err := chainClient.GetChainMeta(ctx, &iotexapi.GetChainMetaRequest{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain meta")
	}
	tipHeight := res.ChainMeta.Height
	atomic.StoreUint64(&_tipHeight, tipHeight)

	daoHeight, err = srv.dao.Height()
	if err != nil {
		return errors.Wrap(err, "failed to get tip height from block dao")
	}
	serverMetrics.WithLabelValues("rpc", "tipHeight").Set(float64(tipHeight))
	serverMetrics.WithLabelValues("db", "daoHeight").Set(float64(daoHeight))
	if config.Default.Iotex.CatchUpMode && daoHeight <= 0 {
		if config.Default.Iotex.CatchUpStartHeight > 0 {
			daoHeight = config.Default.Iotex.CatchUpStartHeight - 1
		} else {
			daoHeight = tipHeight - 1
		}
	}

	srv.logger.Info("start rebuild blockdao",
		zap.Uint64("daoHeight", daoHeight),
		zap.Uint64("tipHeight", tipHeight),
	)
	for startHeight := daoHeight + 1; startHeight <= tipHeight; {
		count := config.Default.Iotex.BatchSize
		if count > tipHeight-startHeight+1 {
			count = tipHeight - startHeight + 1
		}
		rawRequest := &iotexapi.GetRawBlocksRequest{
			StartHeight:         startHeight,
			Count:               count,
			WithReceipts:        true,
			WithTransactionLogs: true,
		}
		srv.logger.Debug("chain client get raw blocks start",
			zap.Uint64("startHeight", startHeight),
			zap.Uint64("count", count),
		)
		timeStart := time.Now()
		response, err := chainClient.GetRawBlocks(context.Background(), rawRequest)
		if err != nil {
			return errors.Wrap(err, "failed to get raw blocks from the chain")
		}
		srv.logger.Debug("chain client get raw blocks end",
			zap.Duration("timeSpent", time.Since(timeStart)),
			zap.Int("blocks", len(response.GetBlocks())),
		)
		for _, blkInfo := range response.GetBlocks() {
			deser := block.NewDeserializer(config.EVMNetworkID())
			blk, err := deser.FromBlockProto(blkInfo.GetBlock())
			if err != nil {
				return err
			}
			receipts := make(map[hash.Hash256]*action.Receipt)
			for _, receiptPb := range blkInfo.GetReceipts() {
				receipt := &action.Receipt{}
				receipt.ConvertFromReceiptPb(receiptPb)
				receipts[receipt.ActionHash] = receipt
				blk.Receipts = append(blk.Receipts, receipt)
			}
			for _, tlogs := range blkInfo.GetTransactionLogs().GetLogs() {
				logs := make([]*action.TransactionLog, len(tlogs.GetTransactions()))
				for i, txn := range tlogs.GetTransactions() {
					amount, ok := new(big.Int).SetString(txn.Amount, 10)
					if !ok {
						return errors.Errorf("failed to parse %s", txn.Amount)
					}
					logs[i] = &action.TransactionLog{
						Type:      txn.Type,
						Amount:    amount,
						Sender:    txn.Sender,
						Recipient: txn.Recipient,
					}
				}
				actHash := hash.BytesToHash256(tlogs.ActionHash)
				receipts[actHash].AddTransactionLogs(logs...)
			}
			if err := srv.dao.PutBlock(ctx, blk); err != nil {
				if errors.Is(err, blockdao.ErrRemoteHeightTooLow) {
					// ignore errors when remote height is too low
					return nil
				}
				return errors.Wrap(err, "failed to build index for the block")
			}
			atomic.StoreUint64(&_daoHeight, blk.Height())
			serverMetrics.WithLabelValues("db", "daoHeight").Set(float64(blk.Height()))
			srv.logger.Debug("putblock to dao", zap.Uint64("blkHeight", blk.Height()))
		}
		startHeight += count
	}
	return nil
}

func (srv *Server) startHTTPService() error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := httputil.Server(config.Default.Server.Http, mux)
	ln, err := httputil.LimitListener(server.Addr)
	if err != nil {
		return err
	}
	if err := server.Serve(ln); err != nil {
		return err
	}
	return nil
}

func (srv *Server) startDaoService(ctx context.Context) error {
	var tip protocol.TipInfo
	ctxDao := protocol.WithBlockchainCtx(
		genesis.WithGenesisContext(context.Background(), genesis.Default),
		protocol.BlockchainCtx{
			Tip: tip,
		},
	)
	var bdao kernel.BatchBlockDao
	var err error
	if config.Default.Iotex.CatchUpMode {
		srv.logger.Warn("currently in catch-up mode, it will be rebuild dao service in momery")
		dao := kernel.NewVirtualDao()
		bdao = kernel.NewLocalBatchBlockDao(dao)
	} else {
		deser := block.NewDeserializer(config.EVMNetworkID())
		path := config.Default.BlockDB.DbPath
		uri, err := url.Parse(path)
		if err != nil {
			return errors.Wrapf(err, "failed to parse chain db path %s", path)
		}

		switch config.Default.BlockDAOProvider {
		case "grpc":
			insec := uri.Query().Get("insecure") == "true"
			fdao := blockdao.NewGrpcBlockDAO(uri.Host, insec, deser)
			dao := blockdao.NewBlockDAOWithIndexersAndCache(fdao, nil, 100)
			opts := []grpc.DialOption{}
			if insec {
				opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
			} else {
				opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})))
			}
			maxRecvSize := config.Default.Iotex.MaxCallRecvMsgSize
			if maxRecvSize > 0 {
				opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxRecvSize)))
			}
			conn, err := grpc.NewClient(uri.Host, opts...)
			if err != nil {
				return err
			}
			cli := iotexapi.NewAPIServiceClient(conn)
			bdao = kernel.NewBatchBlockDao(dao, cli)

		case "p2p":
			svr, err := itx.NewServer(config.Default.ChainConfig)
			if err != nil {
				return errors.Wrapf(err, "failed to create chain server")
			}

			cs := svr.ChainService(config.Default.ChainConfig.Chain.EVMNetworkID)
			dao := cs.BlockDAO()
			bdao = kernel.NewLocalBatchBlockDao(dao)

		default:
			dbConfig := config.Default.BlockDB
			dbConfig.DbPath = uri.Path
			fdao, err := filedao.NewFileDAO(dbConfig.Config, deser)
			if err != nil {
				return errors.Wrapf(err, "failed to create file dao with path %s", uri.Path)
			}
			dao := blockdao.NewBlockDAOWithIndexersAndCache(fdao, nil, 100)
			bdao = kernel.NewLocalBatchBlockDao(dao)
		}
	}
	if err := bdao.Start(ctxDao); err != nil {
		return err
	}
	daoHeight, err := bdao.Height()
	if err != nil {
		return err
	}
	srv.logger.Info("successfully to loaded BlockDAO", zap.Uint64("daoHeight", daoHeight))
	srv.dao = bdao
	go srv.startDaoWorker(ctxDao)
	return nil
}

func (srv *Server) startDaoWorker(ctx context.Context) {
	for {
		if !srv.isRunning.Get() {
			break
		}
		if config.Default.Iotex.CatchUpMode || !config.Default.Iotex.DisableRebuildDB {
			if err := srv.startRebuildBlockDaoWorker(ctx); err != nil {
				srv.logger.Error("failed to start http service", zap.Error(err))
			}
		}
		time.Sleep(time.Second * 4)
	}
}

func (srv *Server) startRPCService(ctx context.Context) error {
	sockAddr := config.Default.Server.Addr
	os.Remove(sockAddr)
	unixAddr, err := net.ResolveUnixAddr("unix", sockAddr)
	if err != nil {
		return err
	}

	listener, err := net.ListenUnix("unix", unixAddr)
	if err != nil {
		return err
	}

	service := NewService(srv.dao)
	for _, pluginFile := range config.Default.Server.Plugins {
		pluginArgs := &Args{Path: pluginFile}
		pluginReply := &Reply{}
		if err := service.Load(pluginArgs, pluginReply); err != nil {
			return err
		}
	}
	if err := service.Start(ctx); err != nil {
		return err
	}
	if err := rpc.Register(service); err != nil {
		return err
	}
	srv.m.Lock()
	srv.service = service
	srv.m.Unlock()
	rpc.Accept(listener)
	return nil
}
