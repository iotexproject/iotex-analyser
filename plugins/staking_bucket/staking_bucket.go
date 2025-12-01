package main

import (
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-analyser/service/staking_bucket"
)

type stakingBucketPlugin struct {
	*staking_bucket.StakingBucketPlugin
}

func (b stakingBucketPlugin) DependentPlugins() []string {
	return []string{"candidate"}
}

// exported
var Plugin = stakingBucketPlugin{
	&staking_bucket.StakingBucketPlugin{
		PluginShadow: plugin.PluginSelf,
	},
}
