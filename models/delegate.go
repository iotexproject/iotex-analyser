package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Delegate struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	OperatorAddress string          `gorm:"size:42;not null;default:'';"`
	RewardAddress   string          `gorm:"size:42;not null;default:'';"`
	OwnerAddress    string          `gorm:"size:42;not null;default:'';uniqueIndex"`
	Candidate       string          `gorm:"size:42;not null;default:'';"`
	Active          bool            `gorm:"type:bool;not null;default:false"`
	Name            string          `gorm:"size:42;not null;default:'';"`
	Productivity    int             `gorm:"type:int2;unsigned;not null;default:0"`
	StakeAmount     decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	VoteWeight      decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	Probated        bool            `gorm:"type:bool;not null;default:false"`
	SelfStake       bool
}

func (Delegate) TableName() string {
	return "delegate"
}

type DelegateRecord struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	OperatorAddress string          `gorm:"size:42;not null;default:'';"`
	RewardAddress   string          `gorm:"size:42;not null;default:'';"`
	OwnerAddress    string          `gorm:"size:42;not null;default:'';"`
	Candidate       string          `gorm:"size:42;not null;default:'';"`
	Active          bool            `gorm:"type:bool;not null;default:false"`
	Name            string          `gorm:"size:42;not null;default:'';"`
	Productivity    int             `gorm:"type:int2;unsigned;not null;default:0"`
	StakeAmount     decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	VoteWeight      decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	Probated        bool            `gorm:"type:bool;not null;default:false"`
	SelfStake       bool
	Timestamp       time.Time `gorm:"type:timestamp;index:;default:CURRENT_TIMESTAMP"`
}

func (DelegateRecord) TableName() string {
	return "delegate_record"
}
