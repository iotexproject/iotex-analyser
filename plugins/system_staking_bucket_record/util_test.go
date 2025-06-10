package main

import (
	"math/big"
	"testing"
	"time"

	"github.com/iotexproject/iotex-core/v2/action/protocol/staking"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
	"github.com/stretchr/testify/require"
)

func TestNFTBuckets(t *testing.T) {
	r := require.New(t)
	//https://iotex.larksuite.com/docx/DlcPdPz8HoiaurxpGc4uydDgsZf
	tests := []struct {
		Amount        string
		AutoStake     bool
		DurationDays  uint32
		WeightedVotes int64
	}{
		{"1000", true, 2, 1076},
		{"1000", false, 2, 1038},
		{"10000", true, 30, 12245},
		{"10000", false, 30, 11865},
		{"10000", true, 91, 12854},
		{"10000", false, 91, 12474},
		{"100000", true, 91, 128543},
		{"100000", false, 91, 124741},
	}
	for _, test := range tests {
		_ = test
		stakeAmount, _ := big.NewInt(0).SetString(test.Amount, 10)
		voteBucket := &staking.VoteBucket{
			StakedAmount:   stakeAmount,
			AutoStake:      test.AutoStake,
			StakedDuration: time.Duration(test.DurationDays*24) * time.Hour,
		}
		weightVotes := staking.CalculateVoteWeight(genesis.Default.VoteWeightCalConsts, voteBucket, false)
		r.Equal(test.WeightedVotes, weightVotes.Int64())
	}
}
