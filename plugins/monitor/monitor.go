package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-analyser/server"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const (
	VERSION     = "0.0.1"
	LarkWebHook = "https://open.larksuite.com/open-apis/bot/v2/hook/dc465df5-0d94-439d-82cb-0efff87723c1"
)

var lock sync.RWMutex

type plugnStat struct {
	lastHeight uint64
}

type monitorPlugin struct {
	cfg   Config
	stats map[string]plugnStat
}

func (b monitorPlugin) setStats(name string, stat plugnStat) {
	lock.Lock()
	defer lock.Unlock()
	b.stats[name] = stat
}

func (b monitorPlugin) getStats(name string) (plugnStat, bool) {
	lock.RLock()
	defer lock.RUnlock()
	stat, ok := b.stats[name]
	return stat, ok
}

func (b monitorPlugin) Name() string {
	return "monitor"
}

func (b monitorPlugin) Type() plugin.Type {
	return plugin.TypeWorker
}

func (b monitorPlugin) Start(ctx context.Context) error {
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		cfg := &Config{}
		if err := yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			b.cfg = *cfg
			log.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}
	if !b.cfg.Enable || b.cfg.Interval == 0 || b.cfg.LarkWebHook == "" {
		log.L().Warn("monitor plugin is disabled")
		return nil
	}
	go func() {
		for range time.Tick(b.cfg.Interval) {
			stats := server.GetRunnerStats()
			log.L().Info("runner stats", zap.Uint64("daoHeight", stats.Server.DaoHeight), zap.Uint64("tipHeight", stats.Server.TipHeight))
			for _, stat := range stats.Runners {
				b.monitor(stats.Server, stat)
				log.L().Info("plugin status", zap.String("name", stat.Name), zap.Int("pluginType", int(stat.PluginType)), zap.Int("pluginStatus", int(stat.PluginStatus)), zap.Error(stat.Error))
			}
		}

	}()
	return nil
}

func (b monitorPlugin) monitor(ss server.ServerStat, rs server.RunnerStat) {
	//skip non-standard plugin
	if rs.PluginType != plugin.TypeStandard {
		return
	}
	plugHeight, err := db.GetIndexHeight(rs.Name)
	if err != nil {
		log.L().Error("get index height failed", zap.Error(err))
		return
	}
	if rs.PluginStatus == server.PluginStatusStartError {
		b.alert(fmt.Sprintf("Analyser plugin Start failed \nName: **%s** **%s**\nPluginHeight: %d TipHeight: %d DaoHeight: %d\nReason: %s", rs.Name, time.Now().Local().Format(time.DateTime), plugHeight, ss.TipHeight, ss.DaoHeight, rs.Error.Error()))
		return
	} else if rs.PluginStatus == server.PluginStatusPutError {
		b.alert(fmt.Sprintf("Analyser plugin PutBlock failed \nName: **%s** **%s**\nPluginHeight: %d TipHeight: %d DaoHeight: %d\nReason: %s", rs.Name, time.Now().Local().Format(time.DateTime), plugHeight, ss.TipHeight, ss.DaoHeight, rs.Error.Error()))
		return
	}
	stat, ok := b.getStats(rs.Name)
	if !ok {
		b.setStats(rs.Name, plugnStat{lastHeight: plugHeight})
		return
	}
	//if plugin height not update
	if plugHeight == stat.lastHeight {
		b.alert(fmt.Sprintf("Analyser plugin PutBlock failed \nName: **%s** **%s**\nPluginHeight: %d TipHeight: %d DaoHeight: %d\nReason: %s", rs.Name, time.Now().Local().Format(time.DateTime), plugHeight, ss.TipHeight, ss.DaoHeight, "plugin height not update"))
		return
	}
}

// LarkMessageRequest is a struct to store the body for a Lark message
type LarkMessageRequest struct {
	MsgType string  `json:"msg_type"`
	Content Content `json:"content"`
}

// Content is a supporting struct to store message content
type Content struct {
	Text string `json:"text"`
}

type LarkMessageResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (b monitorPlugin) alert(msg string) {
	req := LarkMessageRequest{
		MsgType: "text",
		Content: Content{
			Text: msg,
		},
	}
	payload, _ := json.Marshal(req)
	response, err := kernel.DefaultHTTPClient.Post(LarkWebHook, "application/json", bytes.NewReader(payload))
	if err != nil {
		log.L().Error("lark webhook post failed", zap.Error(err))
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.L().Error("read response body failed", zap.Error(err))
		return
	}
	var res LarkMessageResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.L().Error("unmarshal response body failed", zap.Error(err))
		return
	}
	if res.Code != 0 {
		log.L().Error("lark webhook post failed", zap.String("msg", res.Msg))
	}
}

func (b monitorPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return nil
}

func (b monitorPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b monitorPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = monitorPlugin{
	cfg: Config{
		Interval:    60 * 5,
		LarkWebHook: LarkWebHook,
		Enable:      true,
	},
	stats: make(map[string]plugnStat),
}
