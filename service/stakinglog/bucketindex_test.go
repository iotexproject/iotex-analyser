package stakinglog

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

func word(t *testing.T, h string) hash.Hash256 {
	t.Helper()
	raw, err := hex.DecodeString(h)
	require.NoError(t, err)
	require.Len(t, raw, 32)
	var out hash.Hash256
	copy(out[:], raw)
	return out
}

// Topic words captured verbatim from testnet receipts, so the tests pin the
// wire shape rather than restating the expression the implementation uses.
//
//	createStake       block 46,860,857  action 4f26f735…  Topics[1]=0x11d=285
//	candidateRegister block 23,547,180  action 6cbd2f14…  Topics[1]=0xe9=233
const (
	realCreateStakeTopic0       = "0000000000000000000000000000000000000000006372656174655374616b65"
	realCandidateRegisterTopic0 = "00000000000000000000000000000063616e6469646174655265676973746572"
	realCreateStakeTopic1       = "000000000000000000000000000000000000000000000000000000000000011d"
	realCandidateRegisterTopic1 = "00000000000000000000000000000000000000000000000000000000000000e9"
)

// The candidate whose identifier wedged staking_bucket on testnet. Its low 8
// bytes are 7607279276849607506, the value that reached staking_buckets and
// (37 times over) ClickHouse's staking_actions.
const wedgedCandidate = "io1ed52svvdun2qv8sf2m0xnynuxfaulv6jlww7ur"

const otherAddr = "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02"

func legacyLog(t *testing.T, topic0, topic1 string) *action.Log {
	t.Helper()
	return &action.Log{
		Address: ProtocolAddress,
		Topics: action.Topics{
			word(t, topic0), word(t, topic1),
			hash.BytesToHash256(mustAddr(t, wedgedCandidate).Bytes()),
		},
	}
}

func stakedLog(t *testing.T, idx uint64) *action.Log {
	t.Helper()
	topics, data, err := action.PackStakedEvent(
		mustAddr(t, otherAddr), mustAddr(t, wedgedCandidate), idx, big.NewInt(100), 91, true)
	require.NoError(t, err)
	return &action.Log{Address: ProtocolAddress, Topics: topics, Data: data}
}

func registeredLog(t *testing.T, bls []byte) *action.Log {
	t.Helper()
	cand, other := mustAddr(t, wedgedCandidate), mustAddr(t, otherAddr)
	topics, data, err := action.PackCandidateRegisteredEvent(cand, other, cand, "cand", other, bls)
	require.NoError(t, err)
	return &action.Log{Address: ProtocolAddress, Topics: topics, Data: data}
}

func activatedLog(t *testing.T, idx uint64) *action.Log {
	t.Helper()
	topics, data, err := action.PackCandidateActivatedEvent(mustAddr(t, wedgedCandidate), idx)
	require.NoError(t, err)
	return &action.Log{Address: ProtocolAddress, Topics: topics, Data: data}
}

// TestFourShapes walks the whole matrix from BucketIndex's doc comment. Every
// previous copy of this logic handled a different subset of these four.
func TestFourShapes(t *testing.T) {
	var sentinel hash.Hash256
	for i := 24; i < 32; i++ {
		sentinel[i] = 0xff
	}
	for _, tc := range []struct {
		name  string
		logs  func(t *testing.T) []*action.Log
		want  uint64
		found bool
	}{
		{
			name: "legacy createStake",
			logs: func(t *testing.T) []*action.Log {
				return []*action.Log{legacyLog(t, realCreateStakeTopic0, realCreateStakeTopic1)}
			},
			want: 285, found: true,
		},
		{
			name: "legacy candidateRegister with self-stake",
			logs: func(t *testing.T) []*action.Log {
				return []*action.Log{legacyLog(t, realCandidateRegisterTopic0, realCandidateRegisterTopic1)}
			},
			want: 233, found: true,
		},
		{
			name: "legacy candidateRegister without self-stake",
			logs: func(t *testing.T) []*action.Log {
				return []*action.Log{{
					Address: ProtocolAddress,
					Topics: action.Topics{
						word(t, realCandidateRegisterTopic0), sentinel,
						hash.BytesToHash256(mustAddr(t, wedgedCandidate).Bytes()),
					},
				}}
			},
			want: NoBucket, found: true,
		},
		{
			name: "BLS candidateRegister with self-stake",
			logs: func(t *testing.T) []*action.Log {
				return []*action.Log{
					registeredLog(t, []byte{0x01}), stakedLog(t, 290), activatedLog(t, 290),
				}
			},
			want: 290, found: true,
		},
		{
			name: "BLS candidateRegister without self-stake",
			logs: func(t *testing.T) []*action.Log {
				return []*action.Log{registeredLog(t, []byte{0x01})}
			},
			want: NoBucket, found: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := BucketIndex(tc.logs(t))
			require.Equal(t, tc.found, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

// A real index must beat the CandidateRegistered fallback regardless of the
// order the logs arrive in. The previous implementations relied on Staked
// happening to come second.
func TestRealIndexWinsInEitherOrder(t *testing.T) {
	r := require.New(t)
	forward := []*action.Log{registeredLog(t, []byte{0x01}), stakedLog(t, 290)}
	reversed := []*action.Log{stakedLog(t, 290), registeredLog(t, []byte{0x01})}

	got, ok := BucketIndex(forward)
	r.True(ok)
	r.Equal(uint64(290), got)

	got, ok = BucketIndex(reversed)
	r.True(ok)
	r.Equal(uint64(290), got, "CandidateRegistered must not overwrite a real index")
}

// TestCandidateRegisteredIsNotAnIndex is the regression guard for the original
// bug: reading Topics[1] of a CandidateRegistered log yields the candidate's
// address, which truncated to 64 bits is a plausible-looking bucket id.
func TestCandidateRegisteredIsNotAnIndex(t *testing.T) {
	r := require.New(t)
	l := registeredLog(t, []byte{0x01})

	got, ok := BucketIndex([]*action.Log{l})
	r.True(ok)
	r.Equal(NoBucket, got, "no bucket exists, and that is not the same as no index found")

	// The value every previous copy wrote instead.
	cand := mustAddr(t, wedgedCandidate)
	fromTopic1 := new(big.Int).SetBytes(l.Topics[1][:]).Uint64()
	r.Equal(new(big.Int).SetBytes(cand.Bytes()).Uint64(), fromTopic1)
	r.Equal(uint64(7607279276849607506), fromTopic1,
		"this is the value observed in staking_buckets and ClickHouse staking_actions")
}

// Legacy handlers whose Topics[1] is not a bucket index must not match. Only
// createStake and candidateRegister reach the callers' lookup, so the whitelist
// is those two -- candidateUpdate and candidateTransferOwnership put a
// candidate identifier and a caller address there.
func TestOtherLegacyHandlersAreNotMatched(t *testing.T) {
	r := require.New(t)
	cand := mustAddr(t, wedgedCandidate)
	for _, name := range []string{
		staking.HandleCandidateUpdate,
		"candidateTransferOwnership",
		staking.HandleUnstake,
		staking.HandleRestake,
	} {
		_, ok := BucketIndex([]*action.Log{{
			Address: ProtocolAddress,
			Topics: action.Topics{
				hash.BytesToHash256([]byte(name)),
				hash.BytesToHash256(cand.Bytes()),
			},
		}})
		r.False(ok, "%s must not be read for a bucket index", name)
	}
}

// The literal fixtures have to stay what iotex-core derives, or they would
// drift away from the chain with no test noticing.
func TestLegacyTopicsMatchCoreConstants(t *testing.T) {
	require.Equal(t, word(t, realCreateStakeTopic0),
		hash.BytesToHash256([]byte(staking.HandleCreateStake)))
	require.Equal(t, word(t, realCandidateRegisterTopic0),
		hash.BytesToHash256([]byte(staking.HandleCandidateRegister)))
}

// An address parked in Topics[1] of a whitelisted handler would be a core-side
// change this package cannot follow. Refusing it makes the caller's error fire,
// which is loud; accepting it writes an address as a bucket id, which is not.
func TestWhitelistedHandlerRejectsANonIntegerTopic1(t *testing.T) {
	_, ok := BucketIndex([]*action.Log{{
		Address: ProtocolAddress,
		Topics: action.Topics{
			word(t, realCreateStakeTopic0),
			hash.BytesToHash256(mustAddr(t, wedgedCandidate).Bytes()),
		},
	}})
	require.False(t, ok)
}

func TestForeignAndMalformedLogs(t *testing.T) {
	r := require.New(t)
	good := stakedLog(t, 7)

	_, ok := BucketIndex(nil)
	r.False(ok)

	_, ok = BucketIndex([]*action.Log{nil})
	r.False(ok)

	_, ok = BucketIndex([]*action.Log{{
		Address: "io1someothercontract", Topics: good.Topics, Data: good.Data}})
	r.False(ok, "only the staking protocol's own logs count")

	_, ok = BucketIndex([]*action.Log{{Address: ProtocolAddress}})
	r.False(ok, "no topics at all")

	_, ok = BucketIndex([]*action.Log{{
		Address: ProtocolAddress, Topics: action.Topics{hash.Hash256{}}}})
	r.False(ok, "unknown Topics[0]")

	_, ok = BucketIndex([]*action.Log{{
		Address: ProtocolAddress, Topics: action.Topics{word(t, realCreateStakeTopic0)}}})
	r.False(ok, "legacy log truncated to one topic")

	_, ok = BucketIndex([]*action.Log{{
		Address: ProtocolAddress, Topics: good.Topics, Data: []byte{0x01}}})
	r.False(ok, "undecodable Staked data must not produce an index")
}
