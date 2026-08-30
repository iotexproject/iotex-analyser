package staking_bucket

import (
	"math/big"
	"testing"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/stretchr/testify/require"
)

func mustAddr(t *testing.T, s string) address.Address {
	t.Helper()
	a, err := address.FromString(s)
	require.NoError(t, err)
	return a
}

// The candidate whose identifier wedged the plugin on testnet. Its low 8 bytes
// are 7607279276849607506, which is what landed in staking_buckets.bucket_id.
const wedgedCandidate = "io1ed52svvdun2qv8sf2m0xnynuxfaulv6jlww7ur"

// TestStakedBucketIndexComesFromData pins where the bucket index actually lives.
//
// Staked is the only staking event that carries one, and it carries it in the
// data section: topics are [signature, voter, candidate].
func TestStakedBucketIndexComesFromData(t *testing.T) {
	r := require.New(t)
	voter := mustAddr(t, "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02")
	cand := mustAddr(t, wedgedCandidate)

	topics, data, err := action.PackStakedEvent(voter, cand, 285, big.NewInt(100), 91, true)
	r.NoError(err)

	idx, ok := stakedBucketIndex(&action.Log{
		Address: StakingProtocolAddress,
		Topics:  topics,
		Data:    data,
	})
	r.True(ok)
	r.Equal(uint64(285), idx)

	// The old code read Topics[1]. Show what that would have produced, so the
	// regression is stated rather than implied.
	fromTopic1 := new(big.Int).SetBytes(topics[1][:]).Uint64()
	r.NotEqual(idx, fromTopic1)
	r.Equal(new(big.Int).SetBytes(voter.Bytes()).Uint64(), fromTopic1,
		"Topics[1] of Staked is the voter address, not a bucket index")
}

// TestCandidateRegisteredIsNotABucketEvent is the regression guard.
//
// CandidateRegistered's Topics[1] is the candidate identifier. Read as a bucket
// index it yields the low 64 bits of an address -- a plausible-looking id that
// silently corrupted staking_buckets for months, and wedged the plugin outright
// once one exceeded int64.
func TestCandidateRegisteredIsNotABucketEvent(t *testing.T) {
	r := require.New(t)
	cand := mustAddr(t, wedgedCandidate)
	owner := mustAddr(t, wedgedCandidate)

	other := mustAddr(t, "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02")
	topics, data, err := action.PackCandidateRegisteredEvent(
		cand, other, owner, "cand", other, nil)
	r.NoError(err)

	_, ok := stakedBucketIndex(&action.Log{
		Address: StakingProtocolAddress,
		Topics:  topics,
		Data:    data,
	})
	r.False(ok, "a CandidateRegistered log must not yield a bucket index")

	// What the old code would have written, and why it eventually failed: the
	// value is the candidate address truncated to 64 bits.
	got := new(big.Int).SetBytes(topics[1][:]).Uint64()
	want := new(big.Int).SetBytes(cand.Bytes()).Uint64()
	r.Equal(want, got)
	r.Equal(uint64(7607279276849607506), got,
		"this is the value observed in staking_buckets on testnet")
}

// Logs from other contracts, and malformed ones, must be ignored rather than
// decoded into a bogus index.
func TestStakedBucketIndexRejectsForeignAndMalformedLogs(t *testing.T) {
	r := require.New(t)
	voter := mustAddr(t, "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02")
	cand := mustAddr(t, wedgedCandidate)
	topics, data, err := action.PackStakedEvent(voter, cand, 7, big.NewInt(1), 1, false)
	r.NoError(err)

	_, ok := stakedBucketIndex(nil)
	r.False(ok)

	_, ok = stakedBucketIndex(&action.Log{Address: "io1someothercontract", Topics: topics, Data: data})
	r.False(ok, "only the staking protocol's own logs count")

	_, ok = stakedBucketIndex(&action.Log{Address: StakingProtocolAddress})
	r.False(ok, "no topics at all")

	_, ok = stakedBucketIndex(&action.Log{
		Address: StakingProtocolAddress,
		Topics:  action.Topics{hash.Hash256{}},
		Data:    data,
	})
	r.False(ok, "wrong signature in Topics[0]")

	_, ok = stakedBucketIndex(&action.Log{
		Address: StakingProtocolAddress,
		Topics:  topics,
		Data:    []byte{0x01},
	})
	r.False(ok, "undecodable data must not produce an index")
}
