package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// BlockActionPartition is a standalone model for the partitioned table.
// It intentionally avoids index tags that are unsuitable for partitioned parents.
type BlockActionPartition struct {
	ID                 uint64          `gorm:"column:id" sql:"type:bigint"`
	ActionHash         string          `gorm:"size:64;not null"`
	ActionType         string          `gorm:"size:32;not null"`
	BlockHeight        uint64          `gorm:"unsigned" sql:"type:bigint"`
	Sender             string          `gorm:"size:42;not null;default:''"`
	Recipient          string          `gorm:"size:42;not null;default:''"`
	GasPrice           decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	GasLimit           uint64          `gorm:"type:int4;unsigned;not null;default:0"`
	Nonce              uint64          `gorm:"type:int8;unsigned;not null;default:0"`
	Amount             decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	GasConsumed        uint64          `gorm:"type:int4;unsigned;not null;default:0"`
	ChainID            uint32          `gorm:"type:int4;unsigned;not null;default:0"`
	Encoding           uint32          `gorm:"type:int4;unsigned;not null;default:0"`
	Version            uint32          `gorm:"type:int4;unsigned;not null;default:0"`
	ContractAddress    string          `gorm:"size:42;not null;default:''"`
	Status             uint64          `gorm:"type:int2;unsigned;not null;default:0"`
	ExecutionRevertMsg string          `gorm:"size:255;not null;default:''"`
	Payload            []byte
	Timestamp          time.Time `gorm:"type:timestamp"`
}

func (BlockActionPartition) TableName() string {
	return "block_action_partition"
}
