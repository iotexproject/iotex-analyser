package models

import "github.com/iotexproject/iotex-analyser/db"

type AccountMeta struct {
	ID               uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight      uint64 `gorm:"not null;unsigned;index" sql:"type:bigint"`
	Address          string `gorm:"size:42;not null;default:'';uniqueIndex"`
	IsContract       bool   `gorm:"type:bool;not null;default:false"`
	ContractByteCode []byte
}

func (AccountMeta) TableName() string {
	return "account_meta"
}

func (m *AccountMeta) ByAddress(addr string) error {
	return db.DB().Model(m).Where("address = ?", addr).Take(&m).Error
}
