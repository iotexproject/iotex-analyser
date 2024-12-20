package main

import (
	"strings"
)

var (
	versionSuffix = "_v" + strings.ReplaceAll(VERSION, ".", "_")
)

type Erc1155721Holder struct {
	ID              uint64 `gorm:"primary_key;" sql:"type:bigint"`
	ContractAddress string `gorm:"size:42;not null;default:'';index:,length:9;uniqueIndex:idx_contract_address_holder_token_id_2"`
	Holder          string `gorm:"size:42;not null;default:'';index:,length:9;uniqueIndex:idx_contract_address_holder_token_id_2"`
	ErcType         uint16 `gorm:"index;default:0;"` // 1155 or 721
	TokenID         string `gorm:"not null;default:'';uniqueIndex:idx_contract_address_holder_token_id_2"`
	TokenValue      string `gorm:"not null;default:'';"`
}

func (Erc1155721Holder) TableName() string {
	return "erc1155_721_holders" + versionSuffix
}
