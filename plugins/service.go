package plugins

import (
	"context"
	"plugin"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type pluginStatus int

const (
	PluginStatusUnload pluginStatus = iota
	PluginStatusLoaded
	PluginStatusRunning
)

type Args struct {
	Path string
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
		logger:    log.Logger("pluginsService"),
		pluginMap: make(map[string]*runner),
	}
	return s
}

func (s *Service) Start(ctx context.Context) error {
	createSql := "CREATE TABLE IF NOT EXISTS `index_heights` (" +
		"`name` varchar(128) NOT NULL," +
		"`height` bigint(20) unsigned NOT NULL DEFAULT '0'," +
		"PRIMARY KEY (`name`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return errors.Wrap(err, "failed to start plugin service")
	}

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

func (s *Service) registerPlugin(plugin Plugin) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, found := s.pluginMap[plugin.Name()]
	if found {
		return errors.Errorf("the plugin `%s(%s)` has been registered", plugin.Name(), plugin.Version())
	}
	//load plugin
	runner, err := newRunner(PluginStatusLoaded, plugin, s.dao)
	if err != nil {
		return err
	}
	s.pluginMap[plugin.Name()] = runner
	return nil
}

func (s *Service) deregisterPlugin(plugin Plugin) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, found := s.pluginMap[plugin.Name()]
	if !found {
		return errors.Errorf("the plugin `%s(%s)` has not been registered", plugin.Name(), plugin.Version())
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

func loadPluginFile(path string) (Plugin, error) {
	symbol := "Plugin"
	loadedPlugin, err := plugin.Open(path)

	if err != nil {
		return nil, err
	}
	funcSymbol, err := loadedPlugin.Lookup(symbol)
	if err != nil {
		return nil, errors.Errorf("Can't find '%s' symbol in plugin %s %v", symbol, path, err)
	}

	plugin, ok := funcSymbol.(Plugin)
	if !ok {
		return nil, errors.New("unexpected type from module symbol")
	}
	return plugin, nil
}
