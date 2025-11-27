package main

import (
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-analyser/service/staking_actions"
)

// exported
var Plugin = staking_actions.StakingActionPlugin{
	PluginShadow: plugin.PluginShadow{
		ShadowName: func(s string) string { return s + "_shadow" },
		ShadowTable: func(a any) any {
			if s, ok := a.(*models.StakingActions); ok {
				return &models.StakingActionsPatch{
					StakingActions: s,
				}
			}
			return a
		},
	},
}
