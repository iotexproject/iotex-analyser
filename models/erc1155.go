package models

import (
	"time"
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
	return "erc1155_transfer_batchs"
}

type Erc1155TransferSingle struct {
	ID              uint64    `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash      string    `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Operator        string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Sender          string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Recipient       string    `gorm:"size:42;not null;default:'';index:,length:9"`
	SID             string    `gorm:"column:_id;not null;default:'';"`
	Value           string    `gorm:"not null;default:'';"`
	Timestamp       time.Time `gorm:"type:timestamp;"`
}

func (Erc1155TransferSingle) TableName() string {
	return "erc1155_transfer_singles"
}

type Erc1155URI struct {
	ID              uint64    `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash      string    `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Value           string    `gorm:"size:128;not null;default:'';"`
	SID             string    `gorm:"column:_id;not null;default:'';"`
	Timestamp       time.Time `gorm:"type:timestamp;"`
}

func (Erc1155URI) TableName() string {
	return "erc1155_uris"
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
	return "erc1155_approval_for_alls"
}

type Erc1155721Holder struct {
	ID              uint64 `gorm:"primary_key;" sql:"type:bigint"`
	ContractAddress string `gorm:"size:42;not null;default:'';index:,length:9;uniqueIndex:idx_contract_address_holder_token_id"`
	Holder          string `gorm:"size:42;not null;default:'';index:,length:9;uniqueIndex:idx_contract_address_holder_token_id"`
	ErcType         uint16 `gorm:"index;default:0;"` // 1155 or 721
	TokenID         string `gorm:"not null;default:'';uniqueIndex:idx_contract_address_holder_token_id"`
	TokenValue      string `gorm:"not null;default:'';"`
}

func (Erc1155721Holder) TableName() string {
	return "erc1155_721_holders"
}

type Erc1155721Meta struct {
	ID              uint64 `gorm:"primary_key;" sql:"type:bigint"`
	ContractAddress string `gorm:"size:42;not null;default:'';uniqueIndex:,"`
	ErcType         uint16 `gorm:"index;default:0;"`
	IsSBT           bool   `gorm:"type:bool;not null;default:false"`
}

func (Erc1155721Meta) TableName() string {
	return "erc1155_721_meta"
}
