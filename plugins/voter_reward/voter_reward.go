package main

import (
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-analyser/service/voter_reward"
)

// exported
var Plugin = voter_reward.New(plugin.PluginSelf)
