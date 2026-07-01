package models

import (
	"github.com/shopspring/decimal"
)

// DelegateRewards is the per-candidate cumulative rewards rollup consumed by
// iotex-kit's getDelegateRewards() (hub "All Time" view). It mirrors the
// legacy iotex-analytics `delegate_rewards` table maintained by the
// sync_delegate_rewards() proc; the plugin now rebuilds it on a 24h ticker.
type DelegateRewards struct {
	Candidate       string          `gorm:"size:42;primaryKey;not null"`
	BlockReward     decimal.Decimal `gorm:"type:decimal(60,0);not null;default:0;"`
	EpochReward     decimal.Decimal `gorm:"type:decimal(60,0);not null;default:0;"`
	FoundationBonus decimal.Decimal `gorm:"type:decimal(60,0);not null;default:0;"`
	BurnReward      decimal.Decimal `gorm:"type:decimal(60,0);not null;default:0;"`
}

func (DelegateRewards) TableName() string {
	return "delegate_rewards"
}
