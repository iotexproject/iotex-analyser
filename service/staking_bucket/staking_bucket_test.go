package staking_bucket

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol/staking"
	"github.com/stretchr/testify/require"
)

func mustAddr(t *testing.T, s string) address.Address {
	t.Helper()
	a, err := address.FromString(s)
	require.NoError(t, err)
	return a
}

// topic turns a hex string captured from a real receipt into a topic word.
func topic(t *testing.T, h string) hash.Hash256 {
	t.Helper()
	raw, err := hex.DecodeString(h)
	require.NoError(t, err)
	require.Len(t, raw, 32)
	var out hash.Hash256
	copy(out[:], raw)
	return out
}

// Topic words taken verbatim from testnet receipts, so these tests pin the
// wire shape rather than restating the expression the implementation uses.
//
//	createStake       block 46,860,857  action 4f26f735...
//	candidateRegister block 23,547,180  action 6cbd2f14...
const (
	realCreateStakeTopic0       = "0000000000000000000000000000000000000000006372656174655374616b65"
	realCandidateRegisterTopic0 = "00000000000000000000000000000063616e6469646174655265676973746572"
	// Topics[1] of those two logs: 0x11d = 285 and 0xe9 = 233, matching the
	// bucket_id staking_buckets holds for each.
	realCreateStakeTopic1       = "000000000000000000000000000000000000000000000000000000000000011d"
	realCandidateRegisterTopic1 = "00000000000000000000000000000000000000000000000000000000000000e9"
)

// The candidate whose identifier wedged the plugin on testnet. Its low 8 bytes
// are 7607279276849607506, which is what landed in staking_buckets.bucket_id.
const wedgedCandidate = "io1ed52svvdun2qv8sf2m0xnynuxfaulv6jlww7ur"

// TestLegacyLogsCarryTheIndexInTopic1 is the regression guard for the fix that
// followed #168.
//
// Around eleven staking handlers emit the legacy shape -- handler name in
// Topics[0], then bucket index, then candidate -- and createStake is by far the
// most common action on either chain. #168 narrowed decoding to the Staked
// event, which no legacy log emits, so bucketMap would have missed on every
// stake and the CreateStake branch's hard error would have wedged the plugin
// outright.
func TestLegacyLogsCarryTheIndexInTopic1(t *testing.T) {
	for _, tc := range []struct {
		name           string
		topic0, topic1 string
		want           uint64
	}{
		{"createStake", realCreateStakeTopic0, realCreateStakeTopic1, 285},
		{"candidateRegister", realCandidateRegisterTopic0, realCandidateRegisterTopic1, 233},
	} {
		t.Run(tc.name, func(t *testing.T) {
			idx, ok := bucketIndexFromLog(&action.Log{
				Address: StakingProtocolAddress,
				Topics: action.Topics{
					topic(t, tc.topic0),
					topic(t, tc.topic1),
					hash.BytesToHash256(mustAddr(t, wedgedCandidate).Bytes()),
				},
			})
			require.True(t, ok)
			require.Equal(t, tc.want, idx)
		})
	}
}

// The literal topics above have to stay the ones iotex-core derives, or the
// fixtures would drift away from the chain without any test noticing.
func TestLegacyTopicsMatchCoreConstants(t *testing.T) {
	require.Equal(t, topic(t, realCreateStakeTopic0),
		hash.BytesToHash256([]byte(staking.HandleCreateStake)))
	require.Equal(t, topic(t, realCandidateRegisterTopic0),
		hash.BytesToHash256([]byte(staking.HandleCandidateRegister)))
}

// TestStakedEventCarriesTheIndexInData pins the other shape.
//
// A candidateRegister with a BLS key and a self-stake emits events instead of
// the legacy log, and receiptLog.Build discards the legacy fields entirely when
// it does. Staked is the only one of them carrying a bucket index, in its data
// section: its topics are the signature, the voter and the candidate.
func TestStakedEventCarriesTheIndexInData(t *testing.T) {
	r := require.New(t)
	voter := mustAddr(t, "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02")
	cand := mustAddr(t, wedgedCandidate)

	topics, data, err := action.PackStakedEvent(voter, cand, 290, big.NewInt(100), 91, true)
	r.NoError(err)

	idx, ok := bucketIndexFromLog(&action.Log{
		Address: StakingProtocolAddress,
		Topics:  topics,
		Data:    data,
	})
	r.True(ok)
	r.Equal(uint64(290), idx)

	// What the pre-#168 code read instead, and why six rows were corrupt.
	fromTopic1 := new(big.Int).SetBytes(topics[1][:]).Uint64()
	r.NotEqual(idx, fromTopic1)
	r.Equal(new(big.Int).SetBytes(voter.Bytes()).Uint64(), fromTopic1,
		"Topics[1] of Staked is the voter address, not a bucket index")
}

// TestCandidateRegisteredIsNotABucketEvent is the guard for the original bug.
//
// CandidateRegistered's Topics[1] is the candidate identifier. Read as a bucket
// index it yields the low 64 bits of an address -- a plausible-looking id that
// silently corrupted staking_buckets, and wedged the plugin outright once one
// exceeded int64.
func TestCandidateRegisteredIsNotABucketEvent(t *testing.T) {
	r := require.New(t)
	cand := mustAddr(t, wedgedCandidate)
	other := mustAddr(t, "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02")
	topics, data, err := action.PackCandidateRegisteredEvent(
		cand, other, cand, "cand", other, nil)
	r.NoError(err)

	_, ok := bucketIndexFromLog(&action.Log{
		Address: StakingProtocolAddress,
		Topics:  topics,
		Data:    data,
	})
	r.False(ok, "a CandidateRegistered log must not yield a bucket index")

	// The value the old code wrote, and why it eventually failed.
	got := new(big.Int).SetBytes(topics[1][:]).Uint64()
	r.Equal(new(big.Int).SetBytes(cand.Bytes()).Uint64(), got)
	r.Equal(uint64(7607279276849607506), got,
		"this is the value observed in staking_buckets on testnet")
}

// Legacy handlers whose Topics[1] is not a bucket index must not match. Only
// createStake and candidateRegister reach bucketMap, so the filter is limited
// to those two -- candidateUpdate and candidateTransferOwnership put a
// candidate identifier and a caller address there respectively.
func TestOtherLegacyHandlersAreNotMatched(t *testing.T) {
	r := require.New(t)
	cand := mustAddr(t, wedgedCandidate)
	for _, name := range []string{
		staking.HandleCandidateUpdate,
		"candidateTransferOwnership",
		staking.HandleUnstake,
	} {
		_, ok := bucketIndexFromLog(&action.Log{
			Address: StakingProtocolAddress,
			Topics: action.Topics{
				hash.BytesToHash256([]byte(name)),
				hash.BytesToHash256(cand.Bytes()),
			},
		})
		r.False(ok, "%s must not be read for a bucket index", name)
	}
}

// An address parked in Topics[1] of a matched handler would be a core-side
// change this plugin cannot follow. Refusing it makes the caller's hard error
// fire, which is loud; accepting it would write an address as a bucket id,
// which is what went unnoticed for months.
func TestMatchedHandlerRejectsANonIntegerTopic1(t *testing.T) {
	_, ok := bucketIndexFromLog(&action.Log{
		Address: StakingProtocolAddress,
		Topics: action.Topics{
			topic(t, realCreateStakeTopic0),
			hash.BytesToHash256(mustAddr(t, wedgedCandidate).Bytes()),
		},
	})
	require.False(t, ok)
}

func TestBucketIndexFromLogRejectsForeignAndMalformedLogs(t *testing.T) {
	r := require.New(t)
	voter := mustAddr(t, "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02")
	cand := mustAddr(t, wedgedCandidate)
	topics, data, err := action.PackStakedEvent(voter, cand, 7, big.NewInt(1), 1, false)
	r.NoError(err)

	_, ok := bucketIndexFromLog(nil)
	r.False(ok)

	_, ok = bucketIndexFromLog(&action.Log{Address: "io1someothercontract", Topics: topics, Data: data})
	r.False(ok, "only the staking protocol's own logs count")

	_, ok = bucketIndexFromLog(&action.Log{Address: StakingProtocolAddress})
	r.False(ok, "no topics at all")

	_, ok = bucketIndexFromLog(&action.Log{
		Address: StakingProtocolAddress,
		Topics:  action.Topics{hash.Hash256{}},
	})
	r.False(ok, "unknown Topics[0]")

	_, ok = bucketIndexFromLog(&action.Log{
		Address: StakingProtocolAddress,
		Topics:  action.Topics{topic(t, realCreateStakeTopic0)},
	})
	r.False(ok, "legacy log truncated to one topic")

	_, ok = bucketIndexFromLog(&action.Log{
		Address: StakingProtocolAddress,
		Topics:  topics,
		Data:    []byte{0x01},
	})
	r.False(ok, "undecodable Staked data must not produce an index")
}

// TestNoSelfStakeRegistrationEmitsNoIndex pins the shape that has no bucket at
// all, which neither the pre-#168 code nor #170 handled.
//
// core sets bucketIdx from `act.Amount().Sign() > 0`. With no self-stake there
// is no bucket, and a BLS registration emits only CandidateRegistered -- Staked
// and CandidateActivated sit inside the `if withSelfStake` branch. So no log
// carries an index, and the plugin has to take the sentinel from the action.
//
// Testnet block 36,844,514 (action 523fb707…, amount 0) is exactly this: one
// CandidateRegistered log, and a row that should never have been written.
func TestNoSelfStakeRegistrationEmitsNoIndex(t *testing.T) {
	r := require.New(t)
	cand := mustAddr(t, wedgedCandidate)
	other := mustAddr(t, "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02")
	topics, data, err := action.PackCandidateRegisteredEvent(
		cand, other, cand, "mmmmm", other, []byte{0x01})
	r.NoError(err)

	_, ok := bucketIndexFromLog(&action.Log{
		Address: StakingProtocolAddress,
		Topics:  topics,
		Data:    data,
	})
	r.False(ok, "a self-stake-less registration carries no bucket index anywhere")

	// The legacy shape does carry the sentinel, and it must survive the read so
	// the caller's guard skips the row rather than writing bucket 0.
	var sentinel hash.Hash256
	for i := 24; i < 32; i++ {
		sentinel[i] = 0xff
	}
	idx, ok := bucketIndexFromLog(&action.Log{
		Address: StakingProtocolAddress,
		Topics: action.Topics{
			topic(t, realCandidateRegisterTopic0),
			sentinel,
			hash.BytesToHash256(cand.Bytes()),
		},
	})
	r.True(ok)
	r.Equal(uint64(0xffffffffffffffff), idx,
		"the no-self-stake sentinel must read back intact, not be rejected")
}
