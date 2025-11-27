package main

import (
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-analyser/service/staking_bucket"
)

// exported
var Plugin = staking_bucket.StakingBucketPlugin{
	PluginShadow: plugin.PluginShadow{
		ShadowName: func(s string) string { return s + "_shadow" },
		ShadowTable: func(a plugin.Table) plugin.Table {
			if s, ok := a.(*models.StakingBucket); ok {
				return &models.StakingBucketShadow{
					StakingBucket: s,
				}
			}
			return a
		},
	},
}
