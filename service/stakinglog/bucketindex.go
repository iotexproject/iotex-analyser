// Package stakinglog resolves the native staking protocol's receipt logs.
//
// It exists because three plugins -- staking_bucket, staking_actions and
// staking_actions_ch -- each need the bucket index an action created, each had
// its own copy of the logic, and the copies disagreed. staking_actions was
// fixed for the events shape; staking_actions_ch was not, and silently wrote
// the low 64 bits of a candidate address as a bucket id for every BLS
// registration (37 rows on testnet, and mainnet would have followed the moment
// it activated BLS). staking_bucket was fixed separately and a third way.
//
// One implementation, one set of tests. Topic selectors come from iotex-core's
// own ABI rather than from signature strings copied into a const block, so a
// protocol-side rename surfaces as a decode failure instead of a silent miss.
package stakinglog

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol/staking"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"go.uber.org/zap"
)

// ProtocolAddress is the native staking protocol. A log carrying a matching
// selector from any other emitter is ignored: a user contract is free to emit
// the same topics.
const ProtocolAddress = "io1qnpz47hx5q6r3w876axtrn6yz95d70cjl35r53"

// NoBucket is the index core writes for a registration that creates no bucket
// (candidateNoSelfStakeBucketIndex). Callers must not record a row for it --
// there is no such bucket -- but they do need to tell it apart from "no index
// found", which is a decode failure and has to be loud.
const NoBucket = ^uint64(0)

var (
	stakedTopic             hash.Hash256
	candidateActivatedTopic hash.Hash256
	registeredTopic         hash.Hash256

	stakedEvent abi.Event

	// legacyBucketTopics are the Topics[0] of the legacy-shape logs whose
	// Topics[1] is a bucket index *and* whose action type any of the three
	// callers looks up. All of them consume the map for exactly two action
	// types, CreateStake and CandidateRegister, so the whitelist is these two.
	//
	// Restricting by handler rather than by value is what keeps candidateUpdate
	// and candidateTransferOwnership out: their Topics[1] is a candidate
	// identifier and a caller address respectively, and read as an integer
	// either one is a plausible-looking bucket id.
	legacyBucketTopics map[hash.Hash256]struct{}
)

func init() {
	parsed := action.NativeStakingContractABI()
	if parsed == nil {
		panic("stakinglog: iotex-core exposes no native staking ABI")
	}
	for name, dst := range map[string]*hash.Hash256{
		"Staked":              &stakedTopic,
		"CandidateActivated":  &candidateActivatedTopic,
		"CandidateRegistered": &registeredTopic,
	} {
		ev, ok := parsed.Events[name]
		if !ok {
			panic("stakinglog: event " + name + " absent from the native staking ABI")
		}
		*dst = hash.Hash256(ev.ID)
	}
	stakedEvent = parsed.Events["Staked"]

	// The handler name right-aligned into 32 bytes, which is what
	// hash.BytesToHash256 produces for a post-FbkMigration receipt. Before that
	// fork receiptLog.AddTopics is a no-op, so those logs carry no index at all
	// -- and no native-staking-v2 bucket exists to index yet either.
	legacyBucketTopics = map[hash.Hash256]struct{}{
		hash.BytesToHash256([]byte(staking.HandleCreateStake)):       {},
		hash.BytesToHash256([]byte(staking.HandleCandidateRegister)): {},
	}
}

// BucketIndex returns the bucket index a receipt's staking logs name.
//
// receiptLog.Build emits one of two shapes and never both. The legacy one puts
// the handler name in Topics[0] and the index in Topics[1]; a candidateRegister
// carrying a BLS key emits ABI events instead, and Build then discards the
// legacy fields entirely. Four cases result, and every previous copy of this
// logic handled a different subset of them:
//
//	BLS  self-stake  logs emitted                                  index
//	no   yes         legacy                                        Topics[1]
//	no   no          legacy                                        Topics[1] = NoBucket
//	yes  yes         CandidateRegistered+Staked+CandidateActivated  Staked data
//	yes  no          CandidateRegistered only                       none -> NoBucket
//
// The last row is why CandidateRegistered alone yields NoBucket rather than
// nothing: no log names an index, but the caller still must not treat that as a
// decode failure. Resolution is order-independent -- a real index always wins
// over the CandidateRegistered fallback, whatever order the logs arrive in.
//
// (false) means no staking log named an index and none implied one. That is a
// genuine decode failure, and callers turn it into an error rather than
// recording bucket 0.
func BucketIndex(logs []*action.Log) (uint64, bool) {
	var (
		idx           uint64
		found         bool
		sawRegistered bool
	)
	for _, l := range logs {
		if l == nil || l.Address != ProtocolAddress || len(l.Topics) == 0 {
			continue
		}
		switch l.Topics[0] {
		case stakedTopic, candidateActivatedTopic:
			if v, ok := indexFromEventData(l); ok {
				idx, found = v, true
			}
		case registeredTopic:
			sawRegistered = true
		default:
			if _, ok := legacyBucketTopics[l.Topics[0]]; !ok || len(l.Topics) < 2 {
				continue
			}
			if v, ok := uint64FromTopic(l.Topics[1]); ok {
				idx, found = v, true
			}
		}
	}
	switch {
	case found:
		return idx, true
	case sawRegistered:
		return NoBucket, true
	default:
		return 0, false
	}
}

// indexFromEventData reads the leading uint64 of an event's data section.
//
// Staked and CandidateActivated both declare the index as their first
// non-indexed argument; Staked is unpacked through the ABI so a change to the
// tuple fails here instead of shifting the field silently. CandidateActivated
// carries only that one word, so its leading 32 bytes are read directly.
func indexFromEventData(l *action.Log) (uint64, bool) {
	if l.Topics[0] == stakedTopic {
		args, err := stakedEvent.Inputs.NonIndexed().Unpack(l.Data)
		if err != nil || len(args) == 0 {
			log.L().Warn("stakinglog: cannot decode Staked event", zap.Error(err))
			return 0, false
		}
		v, ok := args[0].(uint64)
		return v, ok
	}
	if len(l.Data) < 32 {
		return 0, false
	}
	var word hash.Hash256
	copy(word[:], l.Data[:32])
	return uint64FromTopic(word)
}

// uint64FromTopic reads a uint64 right-aligned into a 32-byte word.
//
// Everything above the low 8 bytes must be zero. An address parked there would
// fail that, so a core-side change this package cannot follow surfaces as a
// missing index -- which the caller reports loudly -- rather than as an address
// recorded as a bucket id, which is how the original bug went unnoticed for
// months.
func uint64FromTopic(w hash.Hash256) (uint64, bool) {
	for _, b := range w[:24] {
		if b != 0 {
			log.L().Warn("stakinglog: staking log has a non-integer where a bucket index belongs",
				zap.String("word", hex.EncodeToString(w[:])))
			return 0, false
		}
	}
	return binary.BigEndian.Uint64(w[24:]), true
}
