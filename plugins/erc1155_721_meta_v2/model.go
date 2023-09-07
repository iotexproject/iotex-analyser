package main

import "strings"

var (
	versionSuffix = "_v" + strings.ReplaceAll(VERSION, ".", "_")
)

type Erc1155721Meta struct {
	ID              uint64 `gorm:"primary_key;" sql:"type:bigint"`
	ContractAddress string `gorm:"size:42;not null;default:'';uniqueIndex:,"`
	ErcType         uint16 `gorm:"index;default:0;"`
	IsSBT           bool   `gorm:"type:bool;not null;default:false"`
}

func (Erc1155721Meta) TableName() string {
	return "erc1155_721_meta" + versionSuffix
}
