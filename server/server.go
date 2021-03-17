package server

import (
	"context"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/go-pkgs/util/httputil"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugins"
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
	dao           blockdao.BlockDAO
	pluginService *plugins.Service
	logger        *zap.Logger
}

func New() *Server {
	s := &Server{
		logger: log.Logger("server"),
	}
	return s
}

// Start start the server
func (srv *Server) Start(ctx context.Context) error {
	if err := kernel.GetDB().Ping(); err != nil {
		return errors.Wrap(err, "failed to ping DB")
	}

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

	srv.logger.Info("start RPC service")
	if err := srv.startRPCService(ctx); err != nil {
		return errors.Wrap(err, "failed to start RPC service")
	}

	return nil
}

func (srv *Server) Stop(ctx context.Context) error {
	if err := srv.pluginService.Stop(ctx); err != nil {
		return err
	}
	return srv.dao.Stop(ctx)
}

func (srv *Server) startRebuildBlockDaoWorker(ctx context.Context) error {
	chainClient := kernel.ChainClient()
	res, err := chainClient.GetChainMeta(ctx, &iotexapi.GetChainMetaRequest{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain meta")
	}
	tipHeight := res.ChainMeta.Height
	lastHeight, err := srv.dao.Height()
	if err != nil {
		return errors.Wrap(err, "failed to get tip height from block dao")
	}
	srv.logger.Info("start rebuild blockdao",
		zap.Uint64("daoHeight", lastHeight),
		zap.Uint64("tipHeight", tipHeight),
	)
	for startHeight := lastHeight + 1; startHeight <= tipHeight; {
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
	if false {
		dao = blockdao.NewBlockDAOInMemForTest(indexers)
	} else {
		dao = blockdao.NewBlockDAO(indexers, config.Default.BlockDB)
	}
	if err := dao.Start(ctxDao); err != nil {
		return err
	}
	srv.dao = dao
	go func() {
		for {
			if err := srv.startRebuildBlockDaoWorker(ctxDao); err != nil {
				srv.logger.Error("failed to start http service", zap.Error(err))
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

	pluginService := plugins.NewService(srv.dao)
	if err := pluginService.Start(ctx); err != nil {
		return err
	}
	for _, pluginFile := range config.Default.Server.Plugins {
		pluginArgs := &plugins.Args{Path: pluginFile}
		pluginReply := &plugins.Reply{}
		if err := pluginService.Load(pluginArgs, pluginReply); err != nil {
			return err
		}
	}
	if err := rpc.Register(pluginService); err != nil {
		return err
	}
	srv.pluginService = pluginService
	rpc.Accept(listener)
	return nil
}
