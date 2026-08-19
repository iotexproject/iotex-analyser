package models

import "github.com/shopspring/decimal"

// VoterRewardDistribution is one voter's share of one delegate's frozen pool,
// paid by one IIP-59 settlement chunk.
//
// The chain emits this as a single DelegateVoterRewardsDistributed log per
// (delegate, chunk) carrying parallel arrays; a row here is one index of those
// arrays. Flattening at index time is what makes "what did this voter earn"
// and "what did this delegate pay out" both answerable with an ordinary index
// scan instead of a JSON array unnest.
type VoterRewardDistribution struct {
	ID          uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight uint64 `gorm:"not null;unsigned;index" sql:"type:bigint"`
	// EpochNumber is the epoch the paying block sits in — the settlement's
	// continuation chunks span several epochs' worth of blocks in principle,
	// so this is the payment time, not the accrual period.
	EpochNumber uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	// Era is the settlement this payment belongs to, i.e. the accrual period.
	// It comes from the log's first indexed topic, which the event ABI calls
	// `epoch` but the protocol fills with the target era (see
	// rewarding.packDelegateChunkLog). Era, not EpochNumber, is what a user
	// means by "which payout was this".
	Era        uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash string `gorm:"size:64;not null;index:,length:9"`
	// LogIndex plus RowIndex identify the row inside the block: one chunk can
	// emit several logs (one per delegate) and each log carries many voters.
	LogIndex uint32 `gorm:"not null;default:0"`
	RowIndex uint32 `gorm:"not null;default:0"`

	DelegateID   string `gorm:"size:42;not null;default:'';index:,length:9"`
	DelegateName string `gorm:"size:42;not null;default:'';index"`
	VoterAddress string `gorm:"size:42;not null;default:'';index:,length:9"`
	// RecipientAddress is where the money actually landed. It differs from
	// VoterAddress when the voter set a reward destination, and equals it for
	// a compounded payout.
	RecipientAddress string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Amount           decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	// Compounded is the only valid test for "was this share redeposited".
	// CompoundBucketID == 0 is not that test: native bucket index 0 is a real
	// bucket, which is exactly why the event carries a parallel bool array.
	Compounded       bool   `gorm:"type:bool;not null;default:false"`
	CompoundBucketID uint64 `gorm:"unsigned;not null;default:0"`
}

func (VoterRewardDistribution) TableName() string {
	return "voter_reward_distribution"
}

// VoterRewardEra summarises one settlement so that "how much was paid for era
// N" and "how far has the current settlement got" do not require scanning the
// per-voter table.
//
// A settlement is initialised in one block and drained across many, so this
// row is created once and then updated in place as chunks land.
type VoterRewardEra struct {
	Era uint64 `gorm:"primary_key;unsigned" sql:"type:bigint"`
	// FreezeHeight is the height every vote weight in this settlement was
	// evaluated at. It precedes InitHeight by roughly 1.5 epochs, so a UI that
	// shows a voter their current stake next to this payout is comparing two
	// different points in time; surface it.
	FreezeHeight uint64 `gorm:"unsigned;not null;default:0"`
	// FirstChunkAt is the earliest block observed paying out this settlement.
	//
	// It is deliberately not called "init height": the block that opens a
	// settlement is the era-boundary epoch's last block, and it emits nothing
	// this indexer can key on — the first thing observable is the first
	// continuation chunk, one block later. Naming it after the opening block
	// would have the UI point users at a height the settlement did not start
	// at.
	FirstChunkAt uint64 `gorm:"unsigned;not null;default:0;index"`
	// CompletedHeight is 0 while the drain is still running.
	CompletedHeight uint64 `gorm:"unsigned;not null;default:0"`
	// ScanPhase mirrors the protocol's cursor: 0 scans [startVoter, max],
	// 1 scans [min, startVoter), 2 is complete.
	ScanPhase        uint32          `gorm:"not null;default:0"`
	ResumeVoter      string          `gorm:"size:42;not null;default:''"`
	DelegateCount    uint32          `gorm:"not null;default:0"`
	VoterCount       uint64          `gorm:"unsigned;not null;default:0"`
	TotalFrozen      decimal.Decimal `gorm:"type:decimal(60,0);not null;default:0"`
	TotalDistributed decimal.Decimal `gorm:"type:decimal(60,0);not null;default:0"`
	// OverrunResidue is what an EPOCH_DRAIN_OVERRUN log reported: the pool
	// left unpaid when a later era boundary arrived before this settlement
	// finished. The money is not lost — it rolls into the next settlement —
	// but a non-zero value here means the drain could not keep up.
	OverrunResidue decimal.Decimal `gorm:"type:decimal(60,0);not null;default:0"`
	LastChunkAt    uint64          `gorm:"unsigned;not null;default:0"`
}

func (VoterRewardEra) TableName() string {
	return "voter_reward_era"
}

// DelegateRewardConfig records, per delegate, whether IIP-59 routes its voter
// rewards on chain and what commission the settlement froze.
//
// Opt-in is a one-way transition but arrives by two different routes: an
// explicit SetVoterRewardOptIn action, which emits VoterRewardOptInSet, and
// the one-shot Hermes migration at the fork block, which emits nothing at all
// and has to be reconciled from state.
type DelegateRewardConfig struct {
	ID          uint64 `gorm:"primary_key;" sql:"type:bigint"`
	DelegateID  string `gorm:"size:42;not null;default:'';uniqueIndex:idx_delegate_era,priority:1"`
	Era         uint64 `gorm:"unsigned;not null;default:0;uniqueIndex:idx_delegate_era,priority:2"`
	BlockHeight uint64 `gorm:"unsigned;not null;default:0;index"`

	DelegateName string `gorm:"size:42;not null;default:'';index"`
	OptedIn      bool   `gorm:"type:bool;not null;default:false;index"`
	// OptInSource distinguishes the two routes above: "action" or "migration".
	// Empty while the delegate has not opted in.
	OptInSource string `gorm:"size:16;not null;default:''"`
	OptInHeight uint64 `gorm:"unsigned;not null;default:0"`

	FreezeHeight uint64 `gorm:"unsigned;not null;default:0"`
	// Commission is expressed the way the protocol stores it: basis points
	// retained by the delegate, so 10000 means the voters get nothing.
	BlockCommissionBps uint64 `gorm:"unsigned;not null;default:0"`
	EpochCommissionBps uint64 `gorm:"unsigned;not null;default:0"`
	// CommissionConfigured is false when the delegate never set its portions
	// in the DelegateProfile contract. The protocol then freezes it at 100%
	// commission, so its voters receive nothing even though it opted in —
	// the single most confusing state a delegate can be in, and worth
	// surfacing rather than rendering as a zero.
	CommissionConfigured bool            `gorm:"type:bool;not null;default:false"`
	TotalWeight          decimal.Decimal `gorm:"type:decimal(60,0);not null;default:0"`
	// VoterAmountFrozen is this delegate's slice of the settlement: the pool
	// the era froze for its voters. Kept per delegate rather than only as an
	// era total because "delegate X froze N and has paid M of it" is the
	// progress a delegate actually asks about.
	VoterAmountFrozen decimal.Decimal `gorm:"type:decimal(60,0);not null;default:0"`
	// PayoutAddress is where the delegate's own commission goes. Opting in
	// moves it from the candidate's reward address to its owner address.
	PayoutAddress string `gorm:"size:42;not null;default:''"`
}

func (DelegateRewardConfig) TableName() string {
	return "delegate_reward_config"
}

// VoterRewardDestination is the account a voter's direct (non-compounded)
// IIP-59 payouts are sent to. This is the on-chain replacement for registering
// a forwarding address with the off-chain Hermes service.
//
// Every change is kept rather than upserted: the old recipient travels in the
// event, and a payout has to be interpreted against the destination that was
// in force at its height, not the current one.
type VoterRewardDestination struct {
	ID           uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight  uint64 `gorm:"unsigned;not null;index" sql:"type:bigint"`
	ActionHash   string `gorm:"size:64;not null;index:,length:9"`
	VoterAddress string `gorm:"size:42;not null;default:'';index:,length:9"`
	OldRecipient string `gorm:"size:42;not null;default:''"`
	NewRecipient string `gorm:"size:42;not null;default:''"`
}

func (VoterRewardDestination) TableName() string {
	return "voter_reward_destination"
}
