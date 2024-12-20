package main

import (
	"strings"
	"time"
)

var (
	versionSuffix = "_v" + strings.ReplaceAll(VERSION, ".", "_")
)

type Erc721Transfer struct {
	ID              uint64    `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash      string    `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string    `gorm:"size:42;not null;default:'';index:,length:9"`
	TokenId         string    `gorm:"not null;default:'';"`
	Sender          string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Recipient       string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Timestamp       time.Time `gorm:"type:timestamp;"`
}

func (Erc721Transfer) TableName() string {
	return "erc721_transfers" + versionSuffix
}

type Erc721Approval struct {
	ID              uint64    `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash      string    `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Owner           string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Approved        string    `gorm:"size:42;not null;default:'';index:,length:9"`
	TokenId         string    `gorm:"not null;default:'';"`
	Timestamp       time.Time `gorm:"type:timestamp;"`
}

func (Erc721Approval) TableName() string {
	return "erc721_approvals" + versionSuffix
}

type Erc721Holder struct {
	ID              uint64 `gorm:"primary_key;" sql:"type:bigint"`
	ContractAddress string `gorm:"size:42;not null;default:'';index:,length:9"`
	Holder          string `gorm:"size:42;not null;default:'';index:,length:9"`
}

func (Erc721Holder) TableName() string {
	return "erc721_holders" + versionSuffix
}

type Erc721ApprovalForAll struct {
	ID              uint64    `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash      string    `gorm:"size:64;not null;index:,length:9"`
	ContractAddress string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Owner           string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Operator        string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Approved        bool      `gorm:"type:bool;not null;default:false"`
	Timestamp       time.Time `gorm:"type:timestamp;"`
}

func (Erc721ApprovalForAll) TableName() string {
	return "erc721_approval_for_alls" + versionSuffix
}
