package models

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

type BlockAction struct {
	ID                 uint64          `gorm:"primary_key;index:idx_r_at_id,priority:3;" sql:"type:bigint"`
	ActionHash         string          `gorm:"size:64;not null;index:,type:hash"`
	ActionType         string          `gorm:"size:32;not null;index:,type:hash;index:idx_ca_at,priority:2;index:idx_r_at,priority:2;index:idx_r_at_id,priority:2;index:idx_s_at,priority:2;"`
	BlockHeight        uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	Sender             string          `gorm:"size:42;not null;default:'';index:,type:hash;index:idx_s_at,priority:1;"`
	Recipient          string          `gorm:"size:42;not null;default:'';index:,type:hash;index:idx_r_at,priority:1;index:idx_r_at_id,priority:1;"`
	GasPrice           decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	GasLimit           uint64          `gorm:"type:int4;unsigned;not null;default:0"`
	Nonce              uint64          `gorm:"type:int8;unsigned;not null;default:0"`
	Amount             decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	GasConsumed        uint64          `gorm:"type:int4;unsigned;not null;default:0"`
	ChainID            uint32          `gorm:"type:int4;unsigned;not null;default:0"`
	Encoding           uint32          `gorm:"type:int4;unsigned;not null;default:0"`
	Version            uint32          `gorm:"type:int4;unsigned;not null;default:0"`
	ContractAddress    string          `gorm:"size:42;not null;default:'';index:,type:hash;index:idx_ca_at,priority:1;"`
	Status             uint64          `gorm:"type:int2;unsigned;not null;default:0"`
	ExecutionRevertMsg string          `gorm:"size:255;not null;default:''"`
	Payload            []byte
	Timestamp          time.Time `gorm:"type:timestamp;index:"`
}

func (BlockAction) TableName() string {
	return "block_action"
}

type ActionType struct {
	BlockHeight uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	Hash        string `gorm:"primary_key;size:64;not null;index:,type:hash"`
	Type        uint   `gorm:"type:int4;unsigned;not null;default:0"`
	// accesslist tx
	AccessList datatypes.JSON `gorm:"type:jsonb"`
	// dynamic fee tx
	GasTipCap decimal.Decimal `gorm:"type:decimal(42,0)"`
	GasFeeCap decimal.Decimal `gorm:"type:decimal(42,0)"`
	// blob tx
	BlobGas      uint64          `gorm:"type:int8;unsigned"`
	BlobFeeCap   decimal.Decimal `gorm:"type:decimal(42,0)"`
	BlobHashes   datatypes.JSON  `gorm:"type:json"`
	BlobGasPrice decimal.Decimal `gorm:"type:decimal(42,0)"`
}

func (ActionType) TableName() string {
	return "action_type"
}
