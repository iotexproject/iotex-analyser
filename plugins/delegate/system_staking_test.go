package main

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// Stake from all three system staking contracts counts toward the delegate it votes for.
// A v2 bucket migrated to v3 votes for the zero address, so its stake counts once,
// through the v3 bucket (iotexinternal/iotex-servers#346).
func TestAddSystemStakings(t *testing.T) {
	r := require.New(t)
	const (
		delegateAddr = "io1d0lezplsupwuc2n8k5wgs5vx3w7s9dk5lj9gyf"
		otherAddr    = "io1hxj8zqchvz3zv2dlgthhuy5yqfygkadzus6q4q"
		zeroAddr     = "io1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqd39ym7"
	)
	bucket := func(delegate, staked, votes string) *SystemStakingBucket {
		return &SystemStakingBucket{DelegateOwnerAddress: delegate, StakedAmount: staked, VotingPower: votes}
	}
	delegateMap := map[string]*Delegate{
		delegateAddr: {StakeAmount: big.NewInt(100), VoteWeight: big.NewInt(0)},
	}
	totalVotes := big.NewInt(0)

	for _, buckets := range [][]*SystemStakingBucket{
		{bucket(delegateAddr, "10", "11")}, // v1
		{bucket(zeroAddr, "7000", "7000"), // v2, migrated to v3
			bucket(otherAddr, "5", "5")}, // v2, a delegate not in the map
		{bucket(delegateAddr, "7000", "8000")}, // v3, the migrated bucket
	} {
		addSystemStakings(delegateMap, buckets, totalVotes)
	}

	r.Len(delegateMap, 1)
	r.Equal("7110", delegateMap[delegateAddr].StakeAmount.String())
	r.Equal("8011", delegateMap[delegateAddr].VoteWeight.String())
	r.Equal("8011", totalVotes.String())
}
