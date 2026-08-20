package kernel

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/iotexproject/iotex-core/v2/action"
)

// ActionTypeString names an action for the indexes that record a type column.
//
// It lives here because two plugins — block_action and block_action_partition —
// both need it and both used to carry their own copy. When the reward grants
// were split into three types, only one copy was updated, and the API reads the
// other table: the explorer went on showing every IIP-59 voter payout as a
// generic "grantReward" even though the fix was in.
func ActionTypeString(act action.Action) string {
	// The three reward grants share a single Go type and differ only in an
	// unexported field, so reflection alone cannot tell them apart. Collapsing
	// them hides IIP-59 entirely: mid-settlement most system actions are voter
	// chunks, and the ones that fail when a settlement is cut off look
	// identical to an ordinary block reward.
	if g, ok := act.(*action.GrantReward); ok {
		switch g.RewardType() {
		case action.BlockReward:
			return "grantBlockReward"
		case action.EpochReward:
			return "grantEpochReward"
		case action.VoterRewardChunk:
			return "grantVoterRewardChunk"
		}
	}
	// TrimPrefix, not TrimLeft: TrimLeft takes a *cutset*, so it would also eat
	// any leading run of {*, a, c, t, i, o, n, .} from a name outside the
	// action package. Harmless while every type name starts upper-case, but not
	// something to leave in a shared helper.
	return firstLowerCase(strings.TrimPrefix(fmt.Sprintf("%T", act), "*action."))
}

func firstLowerCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}
