package main

import (
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-analyser/service/staking_actions"
)

type stakingActionPlugin struct {
	*staking_actions.StakingActionPlugin
}

func (s stakingActionPlugin) DependentPlugins() []string {
	return []string{"candidate"}
}

// exported
var Plugin = stakingActionPlugin{
	StakingActionPlugin: &staking_actions.StakingActionPlugin{
		PluginShadow: plugin.PluginSelf,
	},
}
