package main

import (
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-analyser/service/staking_bucket"
)

// exported
var Plugin = staking_bucket.StakingBucketPlugin{
	PluginShadow: plugin.PluginSelf,
}
