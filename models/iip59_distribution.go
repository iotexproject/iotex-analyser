package models

import (
	"github.com/shopspring/decimal"
)

// IIP59DelegateDistribution is one DelegateDistributed receipt log: one
// delegate's slice of one block of an IIP-59 voter-reward drain.
//
// A settlement is NOT one row. The drain is chunked across blocks, so one
// delegate's era settlement produces one row per block it was paid in. Group on
// (snapshot_hash, delegate, epoch) to reassemble it -- that is the join key the
// protocol guarantees.
//
// THE TWO AMOUNT COLUMNS AGGREGATE DIFFERENTLY. This is the single easiest way
// to get IIP-59 accounting wrong:
//
//   - total_voter_pool is per-chunk. It is the sum of the payouts in THIS log
//     only (rewarding/voter_reward.go: `TotalVoterPool: safeBig(rows.paid)`,
//     where rows.paid accumulates within one block). SUM it across a
//     settlement's rows to get the era total.
//
//   - total_commission is an era constant, repeated verbatim in every chunk of
//     the same settlement (it comes from the drain cursor's EpochCommission,
//     written once per era). Do NOT sum it -- take any single row's value. A
//     settlement drained over 20 blocks would otherwise report 20x the
//     commission.
//
// The field names do not hint at this, so the distinction lives here.
type IIP59DelegateDistribution struct {
	BlockHeight uint64 `gorm:"not null;unsigned;index;uniqueIndex:idx_iip59_dist_block_delegate_epoch" sql:"type:bigint"`
	ActionHash  string `gorm:"size:64;not null;index:,length:9"`
	EpochNumber uint64 `gorm:"not null;unsigned;index;uniqueIndex:idx_iip59_dist_block_delegate_epoch" sql:"type:bigint"`
	// Delegate is the candidate identifier from Topics[2], not necessarily the
	// owner or operator address.
	Delegate string `gorm:"size:42;not null;index;uniqueIndex:idx_iip59_dist_block_delegate_epoch"`
	// RewardAddress is where the delegate's commission was credited.
	RewardAddress string `gorm:"size:42;not null;index:,length:9"`
	// SnapshotHash joins the chunks of one settlement. Hex, no 0x prefix.
	SnapshotHash string `gorm:"size:64;not null;index"`
	// TotalCommission: era constant, repeated per chunk. Never SUM.
	TotalCommission decimal.Decimal `gorm:"type:decimal(60,0);not null;default:0"`
	// TotalVoterPool: this chunk's payouts only. SUM across the settlement.
	TotalVoterPool decimal.Decimal `gorm:"type:decimal(60,0);not null;default:0"`
	// NumVoters is len(voters) in this log, i.e. the number of
	// IIP59VoterReward rows that share this row's key.
	NumVoters uint32 `gorm:"not null;default:0"`
}

func (IIP59DelegateDistribution) TableName() string {
	return "iip59_delegate_distributions"
}

// IIP59VoterReward is one voter's payout within one DelegateDistributed log --
// the fan-out of the parallel arrays the event carries.
//
// The five arrays are positional: voters[i] pairs with recipients[i],
// amounts[i], compoundBucketIds[i] and compounded[i]. This table is that
// pairing made explicit, one row per i.
//
// Two things a consumer must not get wrong:
//
//   - Voter is who earned it; Recipient is who received it. They differ when
//     the voter set a destination via SetVoterRewardDestination. RECONCILE
//     AGAINST Recipient -- Voter does not tell you where the money went.
//
//   - Compounded is the only valid test for "was this compounded". Native
//     bucket index 0 is a real bucket, so CompoundBucketID == 0 is
//     indistinguishable between "compounded into bucket 0" and "not
//     compounded". Read CompoundBucketID only when Compounded is true.
type IIP59VoterReward struct {
	BlockHeight uint64 `gorm:"not null;unsigned;index;uniqueIndex:idx_iip59_voter_block_delegate_voter" sql:"type:bigint"`
	ActionHash  string `gorm:"size:64;not null;index:,length:9"`
	EpochNumber uint64 `gorm:"not null;unsigned;index" sql:"type:bigint"`
	Delegate    string `gorm:"size:42;not null;index;uniqueIndex:idx_iip59_voter_block_delegate_voter"`
	// SnapshotHash joins back to IIP59DelegateDistribution and groups a
	// settlement's rows across blocks.
	SnapshotHash string `gorm:"size:64;not null;index"`
	// Voter earned the share.
	Voter string `gorm:"size:42;not null;index;uniqueIndex:idx_iip59_voter_block_delegate_voter"`
	// Recipient actually received it; equals Voter unless a destination was set.
	Recipient string          `gorm:"size:42;not null;index"`
	Amount    decimal.Decimal `gorm:"type:decimal(60,0);not null;default:0"`
	// CompoundBucketID is meaningful ONLY when Compounded is true.
	CompoundBucketID uint64 `gorm:"unsigned;type:numeric(20,0)"`
	Compounded       bool   `gorm:"not null;default:false;index"`
}

func (IIP59VoterReward) TableName() string {
	return "iip59_voter_rewards"
}

// IIP59DelegateOptIn is an append-only log of SetVoterRewardOptIn actions --
// a delegate turning protocol-native voter reward distribution on or off.
//
// This is history, not current state: a delegate can flip repeatedly. The
// current setting is the row with the greatest block_height for a candidate.
// It matters operationally because a delegate that has never opted in is not
// paid by the protocol at all and stays on the legacy Hermes path.
//
// These actions are deliberately NOT folded into staking_actions. That table's
// columns are shaped for staking operations (bucket id, amount, duration,
// auto-stake) and neither of the IIP-59 settings actions has a natural home in
// them -- opt_in would have to squat on the auto_stake column and a recipient
// address on the candidate column.
type IIP59DelegateOptIn struct {
	BlockHeight uint64 `gorm:"not null;unsigned;index;uniqueIndex:idx_iip59_optin_block_action" sql:"type:bigint"`
	ActionHash  string `gorm:"size:64;not null;index:,length:9;uniqueIndex:idx_iip59_optin_block_action"`
	// Sender is the account that submitted the action.
	Sender string `gorm:"size:42;not null;index:,length:9"`
	// Candidate is the candidate identifier the setting applies to, decoded
	// from the action's candidateIdentifier bytes.
	Candidate string `gorm:"size:42;not null;index"`
	OptIn     bool   `gorm:"not null;index"`
}

func (IIP59DelegateOptIn) TableName() string {
	return "iip59_delegate_opt_ins"
}

// IIP59VoterDestination is an append-only log of SetVoterRewardDestination
// actions -- a voter redirecting where its reward share is paid.
//
// Same shape as IIP59DelegateOptIn: history, not current state. The current
// destination for a voter is the row with the greatest block_height.
//
// This is what makes IIP59VoterReward.Recipient diverge from .Voter, so any
// reconciliation that assumes rewards land in the voter's own account needs
// this table to explain the difference.
type IIP59VoterDestination struct {
	BlockHeight uint64 `gorm:"not null;unsigned;index;uniqueIndex:idx_iip59_dest_block_action" sql:"type:bigint"`
	ActionHash  string `gorm:"size:64;not null;index:,length:9;uniqueIndex:idx_iip59_dest_block_action"`
	// Voter is the sender: the account whose reward destination is being set.
	Voter string `gorm:"size:42;not null;index"`
	// Recipient is the new destination.
	Recipient string `gorm:"size:42;not null;index:,length:9"`
}

func (IIP59VoterDestination) TableName() string {
	return "iip59_voter_destinations"
}
