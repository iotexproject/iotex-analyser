package kernel

import (
	"testing"

	"github.com/iotexproject/iotex-core/action/protocol/rolldpos"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	"github.com/stretchr/testify/require"
)

func TestProtocolNumSubEpochs(t *testing.T) {

	require := require.New(t)

	height := []uint64{0, 1, 12, 25, 38, 53, 59, 80, 90, 93, 120}

	p1 := rolldpos.NewProtocol(
		genesis.Default.NumCandidateDelegates,
		genesis.Default.Blockchain.NumDelegates,
		15,
		rolldpos.EnableDardanellesSubEpoch(genesis.Default.Blockchain.DardanellesBlockHeight, genesis.Default.Blockchain.DardanellesNumSubEpochs),
	)

	for i := 0; i < len(height); i++ {

		numSubEpochs := NumSubEpochs(height[i])
		require.Equal(p1.NumSubEpochs(height[i]), numSubEpochs)
	}

}

func TestGetEpochNum(t *testing.T) {
	require := require.New(t)

	height := []uint64{0, 1, 12, 25, 38, 53, 59, 80, 90, 93, 120}

	p1 := rolldpos.NewProtocol(
		genesis.Default.NumCandidateDelegates,
		genesis.Default.Blockchain.NumDelegates,
		15,
		rolldpos.EnableDardanellesSubEpoch(genesis.Default.Blockchain.DardanellesBlockHeight, genesis.Default.Blockchain.DardanellesNumSubEpochs),
	)
	for i := 0; i < len(height); i++ {
		epochNum := p1.GetEpochNum(height[i])
		require.Equal(GetEpochNum(height[i]), epochNum)
	}

	tests := []struct {
		epochNum  uint64
		blkHeight uint64
	}{
		{20673, 13068285},
		{3631, 1306828},
	}
	for _, test := range tests {
		require.Equal(GetEpochNum(test.blkHeight), test.epochNum)

	}
}

func TestGetEpochHeight(t *testing.T) {

	require := require.New(t)
	epochNum := []uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	for i := 0; i < len(epochNum); i++ {

		p1 := rolldpos.NewProtocol(
			genesis.Default.NumCandidateDelegates,
			genesis.Default.Blockchain.NumDelegates,
			15,
			rolldpos.EnableDardanellesSubEpoch(genesis.Default.Blockchain.DardanellesBlockHeight, genesis.Default.Blockchain.DardanellesNumSubEpochs),
		)
		epochHeight := GetEpochHeight(epochNum[i])
		require.Equal(p1.GetEpochHeight(epochNum[i]), epochHeight)
	}

}

func TestGetEpochLastBlockHeight(t *testing.T) {
	require := require.New(t)
	p1 := rolldpos.NewProtocol(
		genesis.Default.NumCandidateDelegates,
		genesis.Default.Blockchain.NumDelegates,
		15,
		rolldpos.EnableDardanellesSubEpoch(genesis.Default.Blockchain.DardanellesBlockHeight, genesis.Default.Blockchain.DardanellesNumSubEpochs),
	)

	epochNums := []uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, epochNum := range epochNums {
		height := GetEpochLastBlockHeight(epochNum)
		require.Equal(p1.GetEpochLastBlockHeight(epochNum), height)
	}
}

func TestGetSubEpochNum(t *testing.T) {
	require := require.New(t)
	p1 := rolldpos.NewProtocol(
		genesis.Default.NumCandidateDelegates,
		genesis.Default.Blockchain.NumDelegates,
		15,
		rolldpos.EnableDardanellesSubEpoch(genesis.Default.Blockchain.DardanellesBlockHeight, genesis.Default.Blockchain.DardanellesNumSubEpochs),
	)
	epochHeights := []uint64{0, 1, 12, 25, 38, 53, 59, 80, 90, 93, 120}

	for _, epochHeight := range epochHeights {
		subEpochNum := GetSubEpochNum(epochHeight)
		require.Equal(p1.GetSubEpochNum(epochHeight), subEpochNum)
	}
}
