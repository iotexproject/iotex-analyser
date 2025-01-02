package models

import (
	"time"

	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

type BlockAction struct {
	ID                 uint64          `gorm:"primary_key;" sql:"type:bigint"`
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

func (BlockAction) TableName() string {
	return "block_action"
}

type ActionType struct {
	BlockHeight uint64 `gorm:"unsigned" sql:"type:bigint"`
	Hash        string `gorm:"primary_key;size:64;not null"`
	Type        uint   `gorm:"type:int4;unsigned;not null;default:0"`
	// accesslist tx
	AccessList datatypes.JSON `gorm:"type:jsonb"`
	// dynamic fee tx
	GasTipCap decimal.Decimal `gorm:"type:decimal(42,0)"`
	GasFeeCap decimal.Decimal `gorm:"type:decimal(42,0)"`
	// blob tx
	BlobGas      uint64          `gorm:"type:int8;unsigned"`
	BlobFeeCap   decimal.Decimal `gorm:"type:decimal(42,0)"`
	BlobHashes   pq.StringArray  `gorm:"type:text[]"`
	BlobGasPrice decimal.Decimal `gorm:"type:decimal(42,0)"`
}

func (ActionType) TableName() string {
	return "action_type"
}
