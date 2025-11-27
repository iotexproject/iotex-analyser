package main

import (
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-analyser/service/staking_actions"
)

// exported
var Plugin = staking_actions.StakingActionPlugin{
	PluginShadow: plugin.PluginSelf,
}
