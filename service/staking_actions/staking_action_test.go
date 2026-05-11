package staking_actions

import (
	"encoding/binary"
	"testing"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/stretchr/testify/require"
)

// testCandidateBytes is a fixed 20-byte sequence used as the "candidate" in tests.
// The IoTeX address is derived via address.FromBytes — avoids hardcoding a bech32 string.
var testCandidateBytes = [20]byte{
	0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa,
	0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44,
}

// mustIotexAddr converts 20 bytes to an IoTeX address string, panicking on error.
func mustIotexAddr(b [20]byte) string {
	addr, err := address.FromBytes(b[:])
	if err != nil {
		panic(err)
	}
	return addr.String()
}

// buildTopic encodes 20-byte Ethereum address bytes into a 32-byte ABI topic (left-padded).
func buildTopic(b [20]byte) hash.Hash256 {
	var topic hash.Hash256
	copy(topic[12:], b[:])
	return topic
}

// buildScheduledAtData encodes a uint64 as ABI bytes (32-byte big-endian).
func buildScheduledAtData(v uint64) []byte {
	data := make([]byte, 32)
	binary.BigEndian.PutUint64(data[24:], v)
	return data
}

func makeLog(addr string, topics []hash.Hash256, data []byte) *action.Log {
	return &action.Log{
		Address: addr,
		Topics:  topics,
		Data:    data,
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// iotexAddrFromTopic
// ──────────────────────────────────────────────────────────────────────────────

func TestIotexAddrFromTopic(t *testing.T) {
	topic := buildTopic(testCandidateBytes)
	expectedAddr := mustIotexAddr(testCandidateBytes)

	got, err := iotexAddrFromTopic(topic)
	require.NoError(t, err)
	require.Equal(t, expectedAddr, got.String())
}

func TestIotexAddrFromTopic_ZeroTopic(t *testing.T) {
	var topic hash.Hash256 // all zeros → zero address
	got, err := iotexAddrFromTopic(topic)
	require.NoError(t, err)
	require.NotEmpty(t, got.String())
}

// ──────────────────────────────────────────────────────────────────────────────
// candidateIdentityFromLogs
// ──────────────────────────────────────────────────────────────────────────────

func TestCandidateIdentityFromLogs_Request(t *testing.T) {
	topic1 := buildTopic(testCandidateBytes)
	expectedAddr := mustIotexAddr(testCandidateBytes)

	logs := []*action.Log{
		makeLog(StakingProtocolAddress, []hash.Hash256{topicDeactivationRequested, topic1}, nil),
	}

	got, err := candidateIdentityFromLogs(logs, topicDeactivationRequested, topicDeactivated)
	require.NoError(t, err)
	require.Equal(t, expectedAddr, got)
}

func TestCandidateIdentityFromLogs_Confirm(t *testing.T) {
	topic1 := buildTopic(testCandidateBytes)
	expectedAddr := mustIotexAddr(testCandidateBytes)

	// CandidateDeactivated event (Confirm op)
	logs := []*action.Log{
		makeLog(StakingProtocolAddress, []hash.Hash256{topicDeactivated, topic1}, nil),
	}

	got, err := candidateIdentityFromLogs(logs, topicDeactivationRequested, topicDeactivated)
	require.NoError(t, err)
	require.Equal(t, expectedAddr, got)
}

func TestCandidateIdentityFromLogs_EmptyLogs(t *testing.T) {
	_, err := candidateIdentityFromLogs(nil, topicDeactivationRequested)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestCandidateIdentityFromLogs_WrongContractAddress(t *testing.T) {
	topic1 := buildTopic(testCandidateBytes)

	logs := []*action.Log{
		makeLog("io1qnpz47hx5q6r3w876axtrn6yz95d70wrongaddr", []hash.Hash256{topicDeactivationRequested, topic1}, nil),
	}

	_, err := candidateIdentityFromLogs(logs, topicDeactivationRequested)
	require.Error(t, err)
}

func TestCandidateIdentityFromLogs_TooFewTopics(t *testing.T) {
	logs := []*action.Log{
		makeLog(StakingProtocolAddress, []hash.Hash256{topicDeactivationRequested}, nil), // only 1 topic
	}

	_, err := candidateIdentityFromLogs(logs, topicDeactivationRequested)
	require.Error(t, err)
}

func TestCandidateIdentityFromLogs_NoMatchingTopic(t *testing.T) {
	topic1 := buildTopic(testCandidateBytes)

	// Log has topicDeactivationScheduled but we're looking for Request/Confirm
	logs := []*action.Log{
		makeLog(StakingProtocolAddress, []hash.Hash256{topicDeactivationScheduled, topic1}, nil),
	}

	_, err := candidateIdentityFromLogs(logs, topicDeactivationRequested, topicDeactivated)
	require.Error(t, err)
}

func TestCandidateIdentityFromLogs_SkipsNonMatchingFirst(t *testing.T) {
	topic1 := buildTopic(testCandidateBytes)
	expectedAddr := mustIotexAddr(testCandidateBytes)

	logs := []*action.Log{
		makeLog("io1qnpz47hx5q6r3w876axtrn6yz95d70wrongaddr", []hash.Hash256{topicDeactivationRequested, topic1}, nil),
		makeLog(StakingProtocolAddress, []hash.Hash256{topicDeactivationRequested, topic1}, nil),
	}

	got, err := candidateIdentityFromLogs(logs, topicDeactivationRequested)
	require.NoError(t, err)
	require.Equal(t, expectedAddr, got)
}

// ──────────────────────────────────────────────────────────────────────────────
// scheduledAtFromLogs
// ──────────────────────────────────────────────────────────────────────────────

func TestScheduledAtFromLogs(t *testing.T) {
	topic1 := buildTopic(testCandidateBytes)
	const expectedAt = uint64(99887766)
	data := buildScheduledAtData(expectedAt)

	logs := []*action.Log{
		makeLog(StakingProtocolAddress, []hash.Hash256{topicDeactivationScheduled, topic1}, data),
	}

	got, err := scheduledAtFromLogs(logs)
	require.NoError(t, err)
	require.Equal(t, expectedAt, got)
}

func TestScheduledAtFromLogs_EmptyLogs(t *testing.T) {
	_, err := scheduledAtFromLogs(nil)
	require.Error(t, err)
}

func TestScheduledAtFromLogs_WrongAddress(t *testing.T) {
	topic1 := buildTopic(testCandidateBytes)
	data := buildScheduledAtData(12345)

	logs := []*action.Log{
		makeLog("io1qnpz47hx5q6r3w876axtrn6yz95d70wrongaddr", []hash.Hash256{topicDeactivationScheduled, topic1}, data),
	}

	_, err := scheduledAtFromLogs(logs)
	require.Error(t, err)
}

func TestScheduledAtFromLogs_DataTooShort(t *testing.T) {
	topic1 := buildTopic(testCandidateBytes)

	logs := []*action.Log{
		makeLog(StakingProtocolAddress, []hash.Hash256{topicDeactivationScheduled, topic1}, []byte{0x01, 0x02}),
	}

	_, err := scheduledAtFromLogs(logs)
	require.Error(t, err)
	require.Contains(t, err.Error(), "too short")
}

func TestScheduledAtFromLogs_NoMatchingEvent(t *testing.T) {
	topic1 := buildTopic(testCandidateBytes)
	data := buildScheduledAtData(12345)

	// Wrong event topic
	logs := []*action.Log{
		makeLog(StakingProtocolAddress, []hash.Hash256{topicDeactivationRequested, topic1}, data),
	}

	_, err := scheduledAtFromLogs(logs)
	require.Error(t, err)
}

func TestScheduledAtFromLogs_ZeroValue(t *testing.T) {
	topic1 := buildTopic(testCandidateBytes)
	data := buildScheduledAtData(0)

	logs := []*action.Log{
		makeLog(StakingProtocolAddress, []hash.Hash256{topicDeactivationScheduled, topic1}, data),
	}

	got, err := scheduledAtFromLogs(logs)
	require.NoError(t, err)
	require.Equal(t, uint64(0), got)
}
