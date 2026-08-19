package voter_reward

import (
	"context"
	"fmt"
	"sync"

	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const (
	stakingProtocolID   = "staking"
	readCandidatesLimit = 20000
)

// CandidateIndex maps a candidate identifier to the fields the IIP-59 tables
// denormalise: its name, and the two addresses that reward routing chooses
// between.
//
// Identifier is the key rather than the reward address because IIP-59 events
// are delegate-scoped by identifier, and because opting in moves the payout
// from the reward address to the owner address — a reward-address-keyed map
// silently stops resolving the moment a delegate opts in. That is exactly the
// bug the pre-existing reward_history mapping has.
type CandidateIndex struct {
	mu      sync.RWMutex
	byID    map[string]CandidateInfo
	byOwner map[string]CandidateInfo
	height  uint64
}

// CandidateInfo is the denormalised subset of a candidate record.
type CandidateInfo struct {
	Identifier string
	Name       string
	Owner      string
	Reward     string
	Operator   string
}

// NewCandidateIndex returns an empty index. Refresh populates it.
func NewCandidateIndex() *CandidateIndex {
	return &CandidateIndex{byID: map[string]CandidateInfo{}, byOwner: map[string]CandidateInfo{}}
}

// Height is the chain height the index was last refreshed at.
func (c *CandidateIndex) Height() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.height
}

// Refresh reloads the candidate set at the given height. Callers refresh at
// epoch boundaries; between refreshes a newly registered candidate resolves to
// an empty name rather than a wrong one.
func (c *CandidateIndex) Refresh(ctx context.Context, client iotexapi.APIServiceClient, height uint64) error {
	list, err := allStakingCandidates(ctx, client, height)
	if err != nil {
		return err
	}
	byID := make(map[string]CandidateInfo, len(list))
	byOwner := make(map[string]CandidateInfo, len(list))
	for _, cand := range list {
		info := CandidateInfo{
			Identifier: cand.GetId(),
			Name:       cand.GetName(),
			Owner:      cand.GetOwnerAddress(),
			Reward:     cand.GetRewardAddress(),
			Operator:   cand.GetOperatorAddress(),
		}
		if info.Identifier == "" {
			// Pre-Xingu records have no separate identifier; the owner is it.
			info.Identifier = info.Owner
		}
		byID[info.Identifier] = info
		if info.Owner != "" {
			byOwner[info.Owner] = info
		}
	}
	c.mu.Lock()
	c.byID, c.byOwner, c.height = byID, byOwner, height
	c.mu.Unlock()
	return nil
}

// Lookup resolves a candidate identifier. It falls back to the owner index
// because a pre-Xingu candidate is addressed by its owner in some contexts and
// by a generated identifier in others.
func (c *CandidateIndex) Lookup(identifier string) (CandidateInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if info, ok := c.byID[identifier]; ok {
		return info, true
	}
	info, ok := c.byOwner[identifier]
	return info, ok
}

// All returns a snapshot of every known candidate, used by the fork-block
// reconciliation that has no event to work from.
func (c *CandidateIndex) All() []CandidateInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]CandidateInfo, 0, len(c.byID))
	for _, info := range c.byID {
		out = append(out, info)
	}
	return out
}

func allStakingCandidates(
	ctx context.Context, client iotexapi.APIServiceClient, height uint64,
) ([]*iotextypes.CandidateV2, error) {
	var out []*iotextypes.CandidateV2
	for i := uint32(0); ; i++ {
		page, err := stakingCandidatePage(ctx, client, i*readCandidatesLimit, readCandidatesLimit, height)
		if err != nil {
			return nil, errors.Wrap(err, "read staking candidates")
		}
		out = append(out, page.GetCandidates()...)
		if len(page.GetCandidates()) < readCandidatesLimit {
			return out, nil
		}
	}
}

func stakingCandidatePage(
	ctx context.Context, client iotexapi.APIServiceClient, offset, limit uint32, height uint64,
) (*iotextypes.CandidateListV2, error) {
	methodName, err := proto.Marshal(&iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_CANDIDATES,
	})
	if err != nil {
		return nil, err
	}
	arg, err := proto.Marshal(&iotexapi.ReadStakingDataRequest{
		Request: &iotexapi.ReadStakingDataRequest_Candidates_{
			Candidates: &iotexapi.ReadStakingDataRequest_Candidates{
				Pagination: &iotexapi.PaginationParam{Offset: offset, Limit: limit},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	res, err := client.ReadState(ctx, &iotexapi.ReadStateRequest{
		ProtocolID: []byte(stakingProtocolID),
		MethodName: methodName,
		Arguments:  [][]byte{arg},
		Height:     fmt.Sprintf("%d", height),
	})
	if err != nil {
		return nil, err
	}
	list := &iotextypes.CandidateListV2{}
	if err := proto.Unmarshal(res.GetData(), list); err != nil {
		return nil, errors.Wrap(err, "unmarshal CandidateListV2")
	}
	return list, nil
}
