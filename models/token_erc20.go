package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type TokenErc20 struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash      string          `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Amount          decimal.Decimal `gorm:"type:decimal(64,0);not null;default:0;"`
	Sender          string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Recipient       string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Timestamp       time.Time       `gorm:"type:timestamp;"`
}

type TokenErc20Holder struct {
	ID              uint64 `gorm:"primary_key;" sql:"type:bigint"`
	ContractAddress string `gorm:"size:42;not null;default:'';index:,length:9"`
	Holder          string `gorm:"size:42;not null;default:'';index:,length:9"`
}
