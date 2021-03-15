package plugins

import (
	"context"
	"fmt"
	"plugin"
	"sync"
	"time"

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
	Success bool
	Message string
}

type pluginRunning struct {
	status pluginStatus
	Plugin
}
type Service struct {
	logger    *zap.Logger
	ctx       context.Context
	pluginMap map[string]pluginRunning
	mu        sync.RWMutex
}

func NewService() *Service {
	s := &Service{
		logger:    log.Logger("pluginsService"),
		ctx:       context.Background(),
		pluginMap: make(map[string]pluginRunning),
	}
	go s.run()
	return s
}

func (s *Service) run() {
	refreshTicker := time.NewTicker(time.Second * 5)
	defer refreshTicker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-refreshTicker.C:
			if err := s.pluginRefresh(); err != nil {
				s.logger.Error("Cannot refresh plugin service", zap.Error(err))
			}
		}
	}
}

func (s *Service) pluginRefresh() error {
	var plugins map[string]pluginRunning
	s.mu.RLock()
	plugins = s.pluginMap
	s.mu.RUnlock()
	for name, plugin := range plugins {
		switch plugin.status {
		case PluginStatusUnload:
			if err := plugin.Plugin.Stop(s.ctx); err != nil {
				s.logger.Error("failed to unload plugin", zap.String("name", name), zap.Error(err))
			} else {
				delete(plugins, name)
			}
		case PluginStatusLoaded:
			if err := plugin.Plugin.Start(s.ctx); err != nil {
				s.logger.Error("failed to load plugin", zap.String("name", name), zap.Error(err))
			} else {
				plugin.status = PluginStatusRunning
				s.pluginMap[name] = plugin
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
	//loaded plugin
	s.pluginMap[plugin.Name()] = pluginRunning{
		status: PluginStatusLoaded,
		Plugin: plugin,
	}
	return nil
}

func (s *Service) unRegisterPlugin(plugin Plugin) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, found := s.pluginMap[plugin.Name()]
	if !found {
		return errors.Errorf("the plugin `%s` has not been registered", plugin.Name())
	}
	//unloaded plugin
	s.pluginMap[plugin.Name()] = pluginRunning{
		status: PluginStatusUnload,
		Plugin: v,
	}
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
		return nil, fmt.Errorf("Can't find '%s' symbol in plugin %s %v", symbol, path, err)
	}

	plugin, ok := funcSymbol.(Plugin)
	if !ok {
		return nil, errors.New("unexpected type from module symbol")
	}
	return plugin, nil
}
