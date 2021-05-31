package server

import (
	"bytes"
	"context"
	"math/big"
	"plugin"
	"sync"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	iap "github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"github.com/rodaine/table"
	"go.uber.org/zap"
)

type pluginStatus int

const (
	PluginStatusUnload pluginStatus = iota
	PluginStatusLoaded
	PluginStatusRunning
)

type Args struct {
	Path        string
	BlockHeight uint64
}

type Reply struct {
	Message string
	Success bool
}

type Service struct {
	stop      chan bool
	once      *sync.Once
	dao       blockdao.BlockDAO
	logger    *zap.Logger
	pluginMap map[string]*runner
	mu        sync.RWMutex
}

func NewService(dao blockdao.BlockDAO) *Service {
	s := &Service{
		stop:      make(chan bool, 1),
		once:      new(sync.Once),
		dao:       dao,
		logger:    log.Logger("service"),
		pluginMap: make(map[string]*runner),
	}
	return s
}

func (s *Service) Start(ctx context.Context) error {

	go func() {
		refreshTicker := time.NewTicker(time.Second * 5)
		defer refreshTicker.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-ctx.Done():
				return
			case <-refreshTicker.C:
				s.pluginRefresh(ctx)
			}
		}
	}()
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.once.Do(func() {
		s.logger.Info("stopping plugin service")
		s.stop <- true
	})
	s.mu.RLock()
	plugins := s.pluginMap
	s.mu.RUnlock()
	for _, plugin := range plugins {
		if err := plugin.Stop(ctx); err != nil {
			log.L().Error("failed to stop plugin", zap.Error(err))
		}
	}
	return nil
}

func (s *Service) pluginRefresh(ctx context.Context) {
	s.mu.RLock()
	pluginMap := s.pluginMap
	s.mu.RUnlock()
	plugins := make(map[string]*runner)

	for name, plugin := range pluginMap {
		plugins[name] = plugin
		switch plugin.Status() {
		case PluginStatusUnload:
			if err := plugin.Stop(ctx); err != nil {
				s.logger.Error("failed to unload plugin", zap.String("name", name), zap.Error(err))
			} else {
				delete(plugins, name)
			}
		case PluginStatusLoaded:
			ctx = kernel.WithBlockDAOCtx(ctx, s.dao)
			if err := plugin.Start(ctx); err != nil {
				s.logger.Error("failed to load plugin", zap.String("name", name), zap.Error(err))
			} else {
				plugin.UpdateStatus(PluginStatusRunning)
			}
		}
	}
	s.mu.Lock()
	s.pluginMap = plugins
	s.mu.Unlock()
}

func (s *Service) registerPlugin(plug iap.Adapter) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, found := s.pluginMap[plug.Name()]
	if found {
		return errors.Errorf("the plugin `%s(%s)` has been registered", plug.Name(), plug.Version())
	}
	//load plugin
	runner, err := newRunner(PluginStatusLoaded, plug, s.dao)
	if err != nil {
		return err
	}
	s.pluginMap[plug.Name()] = runner
	return nil
}

func (s *Service) deregisterPlugin(plug iap.Adapter) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, found := s.pluginMap[plug.Name()]
	if !found {
		return errors.Errorf("the plugin `%s(%s)` has not been registered", plug.Name(), plug.Version())
	}
	//unload plugin
	v.UpdateStatus(PluginStatusUnload)
	return nil
}

func (s *Service) Load(args *Args, reply *Reply) error {
	plugin, err := loadPluginFile(args.Path)
	if err != nil {
		return errors.Wrap(err, "failed to load plugin")
	}
	if err := s.registerPlugin(plugin); err != nil {
		return errors.Wrap(err, "failed to register plugin")
	}
	return nil
}

func (s *Service) UnLoad(args *Args, reply *Reply) error {
	plugin, err := loadPluginFile(args.Path)
	if err != nil {
		return errors.Wrap(err, "failed to load plugin")
	}
	if err := s.deregisterPlugin(plugin); err != nil {
		return errors.Wrap(err, "failed to deregister plugin")
	}
	return nil
}

func (s *Service) Info(args *Args, reply *Reply) error {
	s.mu.RLock()
	pluginMap := s.pluginMap
	s.mu.RUnlock()
	showFields := []interface{}{
		"Name",
		"Version",
		"PluginHeight",
		"DaoHeight",
	}
	var b bytes.Buffer
	tbl := table.New(showFields...).WithWriter(&b)
	for _, m := range pluginMap {
		height, _ := db.GetIndexHeight(m.plugin.Name())
		daoHeight, _ := s.dao.Height()
		tbl.AddRow(
			m.plugin.Name(),
			m.plugin.Version(),
			height,
			daoHeight,
		)
	}
	tbl.Print()
	reply.Message = b.String()
	return nil
}

func (s *Service) TraceBlockByHeight(args *Args, reply *Reply) error {
	blkHeight := args.BlockHeight
	blk, err := s.dao.GetBlockByHeight(blkHeight)
	if err != nil {
		return err
	}
	receipts, err := s.dao.GetReceipts(blkHeight)
	if err != nil {
		return err
	}
	blk.Receipts = receipts
	actionReceipts := make(map[hash.Hash256]*action.Receipt, len(receipts))
	for _, receipt := range receipts {
		actionReceipts[receipt.ActionHash] = receipt
	}
	tlogs, err := s.dao.TransactionLogs(blkHeight)
	if err != nil {
		return err
	} else {
		for _, l := range tlogs.Logs {
			logs := make([]*action.TransactionLog, len(l.Transactions))
			for i, txn := range l.Transactions {
				amount, ok := new(big.Int).SetString(txn.Amount, 10)
				if !ok {
					s.logger.Error("failed to parse transaction amount", zap.Any("amount", txn.Amount))
					continue
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
	}
	reply.Message = getBlockString(blk)
	return nil
}

func loadPluginFile(path string) (iap.Adapter, error) {
	symbol := "Plugin"
	loadedPlugin, err := plugin.Open(path)

	if err != nil {
		return nil, err
	}
	funcSymbol, err := loadedPlugin.Lookup(symbol)
	if err != nil {
		return nil, errors.Errorf("Can't find '%s' symbol in plugin %s %v", symbol, path, err)
	}

	adapter, ok := funcSymbol.(iap.Adapter)
	if !ok {
		return nil, errors.New("unexpected type from module symbol")
	}
	return adapter, nil
}
