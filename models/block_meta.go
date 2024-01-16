package models

import "github.com/shopspring/decimal"

type BlockMeta struct {
	BlockHeight             uint64          `gorm:"primary_key;" sql:"type:bigint;"`
	GasConsumed             uint64          `gorm:"type:int4;unsigned;not null;default:0"`
	ProducerName            string          `gorm:"size:42;not null;default:'';"`
	ProducerAddress         string          `gorm:"size:42;not null;default:'';"`
	ExpectedProducerName    string          `gorm:"size:42;not null;default:'';"`
	ExpectedProducerAddress string          `gorm:"size:42;not null;default:'';"`
	BlockReward             decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	EpochReward             decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	FoundationBonus         decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	EpochNum                uint64          `gorm:"unsigned;index"`
	EpochHeight             uint64          `gorm:"unsigned;index"`
	BlockSize               uint64          `gorm:"unsigned;not null;default:0;"`
}

func (BlockMeta) TableName() string {
	return "block_meta"
}

type BlockMetaV2 struct {
	BlockHeight             uint64          `gorm:"primary_key;" sql:"type:bigint;"`
	GasConsumed             uint64          `gorm:"type:int4;unsigned;not null;default:0"`
	ProducerName            string          `gorm:"size:42;not null;default:'';"`
	ProducerAddress         string          `gorm:"size:42;not null;default:'';"` //store owner_address, because operator_address may be changed
	ExpectedProducerName    string          `gorm:"size:42;not null;default:'';"`
	ExpectedProducerAddress string          `gorm:"size:42;not null;default:'';"`
	BlockReward             decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	EpochReward             decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	FoundationBonus         decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	EpochNum                uint64          `gorm:"unsigned;index"`
	EpochHeight             uint64          `gorm:"unsigned;index"`
	BlockSize               uint64          `gorm:"unsigned;not null;default:0;"`
}

func (BlockMetaV2) TableName() string {
	return "block_meta_v2"
}
