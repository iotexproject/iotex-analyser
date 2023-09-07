package main

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

var (
	versionSuffix = "_v" + strings.ReplaceAll(VERSION, ".", "_")
)

type Erc1155TransferBatch struct {
	ID              uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash      string `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string `gorm:"size:42;not null;default:'';index:,length:9"`
	Operator        string `gorm:"size:42;not null;default:'';index:,length:9"`
	Sender          string `gorm:"size:42;not null;default:'';index:,length:9"`
	Recipient       string `gorm:"size:42;not null;default:'';index:,length:9"`
	IDs             []byte
	Values          []byte
	Timestamp       time.Time `gorm:"type:timestamp;"`
}

func (Erc1155TransferBatch) TableName() string {
	return "erc1155_transfer_batchs" + versionSuffix
}

type Erc1155TransferSingle struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash      string          `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Operator        string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Sender          string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Recipient       string          `gorm:"size:42;not null;default:'';index:,length:9"`
	SID             decimal.Decimal `gorm:"column:_id;type:decimal(128,0);not null;default:'0';"`
	Value           decimal.Decimal `gorm:"type:decimal(128,0);not null;default:'0';"`
	Timestamp       time.Time       `gorm:"type:timestamp;"`
}

func (Erc1155TransferSingle) TableName() string {
	return "erc1155_transfer_singles" + versionSuffix
}

type Erc1155URI struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash      string          `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Value           string          `gorm:"size:128;not null;default:'';"`
	SID             decimal.Decimal `gorm:"column:_id;type:decimal(128,0);not null;default:'0';"`
	Timestamp       time.Time       `gorm:"type:timestamp;"`
}

func (Erc1155URI) TableName() string {
	return "erc1155_uris" + versionSuffix
}

type Erc1155ApprovalForAll struct {
	ID              uint64    `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash      string    `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Account         string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Operator        string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Approved        bool      `gorm:"type:bool;not null;default:false"`
	Timestamp       time.Time `gorm:"type:timestamp;"`
}

func (Erc1155ApprovalForAll) TableName() string {
	return "erc1155_approval_for_alls" + versionSuffix
}
