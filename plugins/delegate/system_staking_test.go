package main

import (
	"errors"
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

// The delegate run waits until every system-staking plugin it reads has indexed the
// target height. Otherwise a v2 -> v3 migration indexed by v3 but not yet by v2 shows the
// stake in both the old v2 bucket and the new v3 one.
func TestFirstBehind(t *testing.T) {
	r := require.New(t)
	plugins := []string{"v1", "v2", "v3"}
	heights := map[string]uint64{"v1": 100, "v2": 99, "v3": 100}
	failed := errors.New("db down")
	indexHeight := func(name string) (uint64, error) {
		if name == "broken" {
			return 0, failed
		}
		return heights[name], nil
	}

	name, height, err := firstBehind(plugins, 100, indexHeight)
	r.Equal("v2", name)
	r.EqualValues(99, height)
	r.NoError(err)

	name, _, err = firstBehind(plugins, 99, indexHeight)
	r.Equal("", name)
	r.NoError(err)

	name, _, err = firstBehind([]string{"v1", "broken"}, 50, indexHeight)
	r.Equal("broken", name)
	r.ErrorIs(err, failed)
}
