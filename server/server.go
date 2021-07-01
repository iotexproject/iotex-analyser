package server

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/pprof"
	"net/rpc"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/go-pkgs/util/httputil"
	"github.com/iotexproject/iotex-analyser/apiservice"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/action/protocol"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Server struct {
	m         sync.RWMutex
	dao       blockdao.BlockDAO
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
	_, err := db.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect DB")
	}
	// if err := kernel.GetDB().Ping(); err != nil {
	// 	return errors.Wrap(err, "failed to ping DB")
	// }
	srv.isRunning.Set(true)

	if err := srv.startDaoService(); err != nil {
		return errors.Wrap(err, "failed to start blockdao service")
	}

	if config.Default.Server.Http != "" {
		go func() {
			if err := srv.startHTTPService(); err != nil {
				srv.logger.Fatal("failed to start http service", zap.Error(err))
			}
		}()
	}

	if config.Default.Server.GrpcPort > 0 {
		go func() {
			if err := apiservice.StartGRPCService(ctx); err != nil {
				srv.logger.Fatal("failed to start GRPC service", zap.Error(err))
			}
		}()
	}

	if config.Default.Server.GrpcProxyPort > 0 {
		go func() {
			if err := apiservice.StartGRPCProxyService(); err != nil {
				srv.logger.Fatal("failed to start GRPC HTTP Proxy service", zap.Error(err))
			}
		}()
	}

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
		defer func() {
			if err := adminserv.Shutdown(ctx); err != nil {
				log.L().Error("Error when serving metrics data.", zap.Error(err))
			}
		}()
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
	srv.logger.Info("start RPC service")
	if err := srv.startRPCService(ctx); err != nil {
		return errors.Wrap(err, "failed to start RPC service")
	}
	return nil
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

	daoHeight, err = srv.dao.Height()
	if err != nil {
		return errors.Wrap(err, "failed to get tip height from block dao")
	}
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
			StartHeight:  startHeight,
			Count:        count,
			WithReceipts: true,
		}
		srv.logger.Debug("chain client get raw blocks start",
			zap.Uint64("startHeight", startHeight),
			zap.Uint64("count", count),
		)
		timeStart := time.Now()
		getRawBlocksRes, err := chainClient.GetRawBlocks(context.Background(), rawRequest)
		if err != nil {
			return errors.Wrap(err, "failed to get raw blocks from the chain")
		}
		srv.logger.Debug("chain client get raw blocks end",
			zap.Duration("timeSpent", time.Since(timeStart)),
			zap.Int("blocks", len(getRawBlocksRes.GetBlocks())),
		)
		for _, blkInfo := range getRawBlocksRes.GetBlocks() {
			blk := &block.Block{}
			if err := blk.ConvertFromBlockPb(blkInfo.GetBlock()); err != nil {
				return errors.Wrap(err, "failed to convert block protobuf to raw block")
			}
			receipts := map[hash.Hash256]*action.Receipt{}
			for _, receiptPb := range blkInfo.GetReceipts() {
				receipt := &action.Receipt{}
				receipt.ConvertFromReceiptPb(receiptPb)
				receipts[receipt.ActionHash] = receipt
				blk.Receipts = append(blk.Receipts, receipt)
			}
			transactionLogs, err := chainClient.GetTransactionLogByBlockHeight(
				context.Background(),
				&iotexapi.GetTransactionLogByBlockHeightRequest{
					BlockHeight: blk.Header.Height(),
				},
			)
			if err != nil {
				return errors.Wrap(err, "failed to fetch transaction logs")
			}
			for _, tlogs := range transactionLogs.TransactionLogs.Logs {
				logs := make([]*action.TransactionLog, len(tlogs.Transactions))
				for i, txn := range tlogs.Transactions {
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
				return errors.Wrap(err, "failed to build index for the block")
			}
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

func (srv *Server) startDaoService() error {
	var tip protocol.TipInfo
	ctxDao := protocol.WithBlockchainCtx(
		context.Background(),
		protocol.BlockchainCtx{
			Genesis: genesis.Default,
			Tip:     tip,
		},
	)
	var indexers []blockdao.BlockIndexer
	var dao blockdao.BlockDAO
	if config.Default.Iotex.CatchUpMode {
		srv.logger.Warn("currently in catch-up mode, it will be rebuild dao service in momery")
		dao = kernel.NewVirtualDao()
	} else {
		dao = blockdao.NewBlockDAO(indexers, config.Default.BlockDB)
	}
	if err := dao.Start(ctxDao); err != nil {
		return err
	}
	daoHeight, err := dao.Height()
	if err != nil {
		return err
	}
	srv.logger.Info("successfully to loaded BlockDAO", zap.Uint64("daoHeight", daoHeight))
	srv.dao = dao
	go func() {
		for {
			if !srv.isRunning.Get() {
				break
			}
			if !config.Default.Iotex.DisableRebuildDB {
				if err := srv.startRebuildBlockDaoWorker(ctxDao); err != nil {
					srv.logger.Error("failed to start http service", zap.Error(err))
				}
			}
			time.Sleep(time.Second * 4)
		}
	}()
	return nil
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
	if err := service.Start(ctx); err != nil {
		return err
	}
	for _, pluginFile := range config.Default.Server.Plugins {
		pluginArgs := &Args{Path: pluginFile}
		pluginReply := &Reply{}
		if err := service.Load(pluginArgs, pluginReply); err != nil {
			return err
		}
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
