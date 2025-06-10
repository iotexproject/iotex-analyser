package systemstaking

import (
	"math/big"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/v2/action/protocol/staking"
)

type BucketInfo struct {
	OwnerAddress         string
	DelegateOwnerAddress string
	StakedAmount         string
	Amount               string
	VotingPower          string
	AutoStake            bool
	Duration             uint32
	DurationType         uint8
	CreateTime           int64
	StakeStartTime       int64
	UnstakeStartTime     int64
	Muted                bool
}

func GetVoteWeight(blkHeight uint64, duration time.Duration, stakeAmount *big.Int, autoStake, selfStake bool) *big.Int {
	if blkHeight < config.Default.Genesis.RedseaBlockHeight {
		return stakeAmount
	}
	voteBucket := &staking.VoteBucket{
		StakedAmount:   stakeAmount,
		AutoStake:      autoStake,
		StakedDuration: duration,
	}
	return staking.CalculateVoteWeight(config.Default.Genesis.Staking.VoteWeightCalConsts, voteBucket, selfStake)
}

func BlocksToDuration(blocks uint32, halfBlockInterval bool) time.Duration {
	duration := time.Duration(blocks) * 5 * time.Second
	if halfBlockInterval {
		// 2.5s per block
		duration /= 2
	}
	return duration
}

func DurationByType(duration time.Duration, durationType uint8) uint32 {
	if durationType == 0 {
		// 5s per block in days
		return uint32(duration.Hours() / 24)
	} else if durationType == 1 {
		// 2.5s per block in seconds
		return uint32(duration.Seconds())
	} else if durationType == 2 {
		// seconds in seconds
		return uint32(duration.Seconds())
	}
	return 0
}

func DurationFromType(duration uint32, durationType uint8) time.Duration {
	if durationType == 0 {
		// 5s per block in days
		return time.Duration(duration*24) * time.Hour
	} else if durationType == 1 {
		// 2.5s per block in seconds
		return time.Duration(duration) * time.Second
	} else if durationType == 2 {
		// seconds in seconds
		return time.Duration(duration) * time.Second
	}
	return 0
}
