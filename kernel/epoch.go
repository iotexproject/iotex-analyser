package kernel

import (
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rolldpos"
)

var (
	rolldposProtocol *rolldpos.Protocol
)

// https://github.com/millken/iotex-core/blob/77950cec681d2e441a77b2b9a162ffa1c4ca4f55/action/protocol/rolldpos/epoch.go#L213
// GetEpochNum returns the number of the epoch for a given height
func GetEpochNum(height uint64) uint64 {
	return rolldposProtocol.GetEpochNum(height)
}

// NumSubEpochs returns the number of subEpochs given a block height
func NumSubEpochs(height uint64) uint64 {
	return rolldposProtocol.NumSubEpochs(height)
}

// GetEpochHeight returns the start height of an epoch
func GetEpochHeight(epochNum uint64) uint64 {
	return rolldposProtocol.GetEpochHeight(epochNum)
}

// GetEpochLastBlockHeight returns the last height of an epoch
func GetEpochLastBlockHeight(epochNum uint64) uint64 {
	return rolldposProtocol.GetEpochLastBlockHeight(epochNum)
}

// GetSubEpochNum returns the sub epoch number of a block height
func GetSubEpochNum(height uint64) uint64 {
	return rolldposProtocol.GetSubEpochNum(height)
}

// FairbankEffectiveHeight returns the effective height of fairbank  = 5166361
func FairbankEffectiveHeight() uint64 {
	p := config.Default.Genesis.Blockchain
	return p.FairbankBlockHeight + p.NumDelegates*p.DardanellesNumSubEpochs
}
