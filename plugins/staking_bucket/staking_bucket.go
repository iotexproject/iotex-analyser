package main

import (
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-analyser/service/staking_bucket"
)

type stakingBucketPlugin struct {
	*staking_bucket.StakingBucketPlugin
}

// exported
var Plugin = stakingBucketPlugin{
	&staking_bucket.StakingBucketPlugin{
		PluginShadow: plugin.PluginSelf,
	},
}
