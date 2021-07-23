package kernel

import "github.com/iotexproject/iotex-core/blockchain/genesis"

var genesisCfg = genesis.Default

//https://github.com/millken/iotex-core/blob/77950cec681d2e441a77b2b9a162ffa1c4ca4f55/action/protocol/rolldpos/epoch.go#L213
// GetEpochNum returns the number of the epoch for a given height
func GetEpochNum(height uint64) uint64 {
	if height == 0 {
		return 0
	}
	p := genesisCfg.Blockchain
	if height <= p.DardanellesBlockHeight {
		return (height-1)/p.NumDelegates/p.NumSubEpochs + 1
	}
	dardanellesEpoch := GetEpochNum(p.DardanellesBlockHeight)
	dardanellesEpochHeight := GetEpochHeight(dardanellesEpoch)
	return dardanellesEpoch + (height-dardanellesEpochHeight)/p.NumDelegates/p.DardanellesNumSubEpochs
}

// NumSubEpochs returns the number of subEpochs given a block height
func NumSubEpochs(height uint64) uint64 {
	p := genesisCfg.Blockchain
	if height < p.DardanellesBlockHeight {
		return p.NumSubEpochs
	}
	return p.DardanellesNumSubEpochs
}

// GetEpochHeight returns the start height of an epoch
func GetEpochHeight(epochNum uint64) uint64 {
	if epochNum == 0 {
		return 0
	}
	p := genesisCfg.Blockchain
	dardanellesEpoch := GetEpochNum(p.DardanellesBlockHeight)
	if epochNum <= dardanellesEpoch {
		return (epochNum-1)*p.NumDelegates*p.NumSubEpochs + 1
	}
	dardanellesEpochHeight := GetEpochHeight(dardanellesEpoch)
	return dardanellesEpochHeight + (epochNum-dardanellesEpoch)*p.NumDelegates*p.DardanellesNumSubEpochs
}

// GetEpochLastBlockHeight returns the last height of an epoch
func GetEpochLastBlockHeight(epochNum uint64) uint64 {
	return GetEpochHeight(epochNum+1) - 1
}

// GetSubEpochNum returns the sub epoch number of a block height
func GetSubEpochNum(height uint64) uint64 {
	p := genesisCfg.Blockchain
	return (height - GetEpochHeight(GetEpochNum(height))) / p.NumDelegates
}
