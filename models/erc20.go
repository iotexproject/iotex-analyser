package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Erc20Transfer struct {
	ID              uint64          `gorm:"primary_key;uniqueIndex:idx_ca_id,priority:4;uniqueIndex:idx_ca_s_id,priority:5;uniqueIndex:idx_ca_r_id,priority:5;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash      string          `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string          `gorm:"size:42;not null;default:'';index:,length:9;uniqueIndex:idx_ca_id,priority:3;uniqueIndex:idx_ca_s_id,priority:3;uniqueIndex:idx_ca_r_id,priority:3;"`
	Amount          decimal.Decimal `gorm:"type:decimal(128,0);not null;default:0;"`
	Sender          string          `gorm:"size:42;not null;default:'';index:,length:9;uniqueIndex:idx_ca_s_id,priority:4;"`
	Recipient       string          `gorm:"size:42;not null;default:'';index:,length:9;uniqueIndex:idx_ca_r_id,priority:4;"`
	Timestamp       time.Time       `gorm:"type:timestamp;"`
}

func (Erc20Transfer) TableName() string {
	return "erc20_transfers"
}

type Erc20Approval struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash      string          `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Amount          decimal.Decimal `gorm:"type:decimal(128,0);not null;default:0;"`
	Owner           string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Spender         string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Timestamp       time.Time       `gorm:"type:timestamp;"`
}

func (Erc20Approval) TableName() string {
	return "erc20_approvals"
}

type Erc20Holder struct {
	ID              uint64 `gorm:"primary_key;" sql:"type:bigint"`
	ContractAddress string `gorm:"size:42;not null;default:'';index:,length:9"`
	Holder          string `gorm:"size:42;not null;default:'';index:,length:9"`
}

func (Erc20Holder) TableName() string {
	return "erc20_holders"
}

type Erc20Meta struct {
	ID              uint64 `gorm:"primary_key;" sql:"type:bigint"`
	ContractAddress string `gorm:"size:42;not null;default:'';index:,length:9"`
	Name            string `gorm:"size:255;not null;default:'';"`
	Symbol          string `gorm:"size:255;not null;default:'';"`
	Decimals        int    `gorm:"not null;default:0;"`
}

func (Erc20Meta) TableName() string {
	return "erc20_metas"
}
