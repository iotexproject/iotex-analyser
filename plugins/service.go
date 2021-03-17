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
	dao       blockdao.BlockDAO
	ctx       context.Context
	logger    *zap.Logger
	pluginMap map[string]*runner
	mu        sync.RWMutex
}

func NewService(ctx context.Context, dao blockdao.BlockDAO) *Service {
	s := &Service{
		dao:       dao,
		logger:    log.Logger("pluginsService"),
		ctx:       ctx,
		pluginMap: make(map[string]*runner),
	}
	go s.run(ctx)
	return s
}

func (s *Service) run(ctx context.Context) {

	createSql := "CREATE TABLE IF NOT EXISTS `index_heights` (" +
		"`name` varchar(128) NOT NULL," +
		"`height` bigint(20) unsigned NOT NULL DEFAULT '0'," +
		"PRIMARY KEY (`name`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		s.logger.Fatal("failed to create index_heights table", zap.Error(err))
	}

	refreshTicker := time.NewTicker(time.Second * 5)
	defer refreshTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.stop()
			return
		case <-refreshTicker.C:
			if err := s.pluginRefresh(); err != nil {
				s.logger.Error("Cannot refresh plugin service", zap.Error(err))
			}
		}
	}
}

func (s *Service) stop() error {
	var plugins map[string]*runner
	s.mu.RLock()
	plugins = s.pluginMap
	s.mu.RUnlock()
	for _, plugin := range plugins {
		if err := plugin.Stop(s.ctx); err != nil {
			log.L().Error("failed to stop plugin", zap.Error(err))
		}
	}
	return nil
}

func (s *Service) pluginRefresh() error {
	var plugins map[string]*runner
	s.mu.RLock()
	plugins = s.pluginMap
	s.mu.RUnlock()
	for name, plugin := range plugins {
		switch plugin.Status() {
		case PluginStatusUnload:
			if err := plugin.Stop(s.ctx); err != nil {
				s.logger.Error("failed to unload plugin", zap.String("name", name), zap.Error(err))
			} else {
				delete(plugins, name)
			}
		case PluginStatusLoaded:
			if err := plugin.Start(s.ctx); err != nil {
				s.logger.Error("failed to load plugin", zap.String("name", name), zap.Error(err))
			} else {
				plugin.UpdateStatus(PluginStatusRunning)
			}
		}
	}
	s.mu.Lock()
	s.pluginMap = plugins
	s.mu.Unlock()
	return nil
}

func (s *Service) registerPlugin(plugin Plugin) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, found := s.pluginMap[plugin.Name()]
	if found {
		return errors.Errorf("the plugin `%s` has been registered", plugin.Name())
	}
	//load plugin
	runner, err := newRunner(PluginStatusLoaded, plugin, s.dao)
	if err != nil {
		return err
	}
	s.pluginMap[plugin.Name()] = runner
	return nil
}

func (s *Service) unRegisterPlugin(plugin Plugin) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, found := s.pluginMap[plugin.Name()]
	if !found {
		return errors.Errorf("the plugin `%s` has not been registered", plugin.Name())
	}
	//unload plugin
	v.UpdateStatus(PluginStatusUnload)
	return nil
}

func (s *Service) Load(args *Args, reply *Reply) error {
	plugin, err := LoadPlugin(args.Path, "Plugin")
	if err != nil {
		return errors.Wrap(err, "failed to load plugin")
	}
	if err := s.registerPlugin(plugin); err != nil {
		return errors.Wrap(err, "failed to register plugin")
	}
	return nil
}

func (s *Service) UnLoad(args *Args, reply *Reply) error {
	plugin, err := LoadPlugin(args.Path, "Plugin")
	if err != nil {
		return errors.Wrap(err, "failed to load plugin")
	}
	if err := s.unRegisterPlugin(plugin); err != nil {
		return errors.Wrap(err, "failed to unregister plugin")
	}
	return nil
}

func LoadPlugin(path string, symbol string) (Plugin, error) {
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
