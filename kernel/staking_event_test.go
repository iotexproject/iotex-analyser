package kernel

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/stretchr/testify/require"
)

func TestNativeStakingEvent(t *testing.T) {
	type logCase struct {
		address string
		topics  []string
		data    string
	}
	cases := []struct {
		name                  string
		logs                  []logCase
		postFairbankMigration bool
		postBLS               bool
		expected              *CandidateRegisterEvent
	}{
		{
			name:     "empty_logs",
			logs:     []logCase{},
			expected: nil,
		},
		{
			name: "not_staking_protocol_address",
			logs: []logCase{
				{
					address: "0x0000000000000000000000000000000000000000",
					topics:  []string{},
					data:    "",
				},
			},
			expected: nil,
		},
		{
			name: "not_candidate_register_topic",
			logs: []logCase{
				{
					address: stakingProtocolAddress.Hex(),
					topics: []string{
						"0x00000000000000000000000009e50c45790be9020e7d22fe6fdc61c5f980c191",
					},
					data: "",
				},
			},
			expected: nil,
		},
		{
			name: "valid_candidate_register_post_fairbank",
			logs: []logCase{
				{
					address: "0x04c22afae6a03438b8fed74cb1cf441168df3f12",
					topics: []string{
						"0x00000000000000000000000000000063616e6469646174655265676973746572",
						"0x000000000000000000000000000000000000000000000000000000000000f417",
						"0x00000000000000000000000009e50c45790be9020e7d22fe6fdc61c5f980c191",
					},
					data: "",
				},
			},
			postFairbankMigration: true,
			expected: &CandidateRegisterEvent{
				BucketID: 62487,
				Candidate: func() address.Address {
					addr, _ := address.FromString("io1p8jsc3tep05syrnaytlxlhrpchucpsv30vwqwl")
					return addr
				}(),
			},
		},
		{
			name: "valid_candidate_register_post_bls",
			logs: []logCase{
				{
					address: "0x04c22afae6a03438b8fed74cb1cf441168df3f12",
					topics: []string{
						"0x48abaaf82f447d8dd05ea2007853e813ff0868d0db4505dae2d2211e080503d9",
						"0x000000000000000000000000cb68a8318de4d4061e0956de69927c327bcfb352",
						"0x000000000000000000000000cb68a8318de4d4061e0956de69927c327bcfb352",
					},
					data: "0x000000000000000000000000cb68a8318de4d4061e0956de69927c327bcfb3520000000000000000000000000000000000000000000000000000000000000080000000000000000000000000cb68a8318de4d4061e0956de69927c327bcfb35200000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000056d6d6d6d6d0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000030ad434f12f30f2d38eb7671c3e19be0e5c74416c71b9298ae39c680719f27502da61e17397eceef112f9ae50bfd7f398e00000000000000000000000000000000",
				},
				{
					address: "0x04c22afae6a03438b8fed74cb1cf441168df3f12",
					topics: []string{
						"0x22e58ff0ddbb59c2141e87863c3a83f087698bb21b388be5dce2b696bbd2e5f3",
						"0x000000000000000000000000cb68a8318de4d4061e0956de69927c327bcfb352",
						"0x000000000000000000000000cb68a8318de4d4061e0956de69927c327bcfb352",
					},
					data: "0x000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000fe1c215e8f838e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				},
				{
					address: "0x04c22afae6a03438b8fed74cb1cf441168df3f12",
					topics: []string{
						"0x3490b26e9297d88e0807bdfc3a04e61c70f42b871566c86a4453428690956171",
						"0x000000000000000000000000cb68a8318de4d4061e0956de69927c327bcfb352",
					},
					data: "0x0000000000000000000000000000000000000000000000000000000000000001",
				},
			},
			postFairbankMigration: true,
			postBLS:               true,
			expected: &CandidateRegisterEvent{
				BucketID: 1,
				Candidate: func() address.Address {
					addr, _ := address.FromString("io1ed52svvdun2qv8sf2m0xnynuxfaulv6jlww7ur")
					return addr
				}(),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := require.New(t)
			logs := make([]*action.Log, 0)
			for _, l := range c.logs {
				topics := make([]hash.Hash256, 0)
				for _, tStr := range l.topics {
					data, err := hex.DecodeString(strings.TrimPrefix(tStr, "0x"))
					r.NoError(err, "failed to decode topic string: %s", tStr)
					topics = append(topics, hash.BytesToHash256(data))
				}
				data, err := hex.DecodeString(strings.TrimPrefix(l.data, "0x"))
				r.NoError(err)
				addrData, err := hex.DecodeString(strings.TrimPrefix(l.address, "0x"))
				r.NoError(err)
				addr, err := address.FromBytes(addrData)
				r.NoError(err)
				logs = append(logs, &action.Log{
					Address: addr.String(),
					Topics:  topics,
					Data:    data,
				})
			}
			event, err := ParseCandidateRegisterEvent(logs, c.postFairbankMigration, c.postBLS)
			r.NoError(err)
			r.Equal(c.expected, event)
		})
	}
}
