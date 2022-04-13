package models

import (
	"github.com/iotexproject/iotex-analyser/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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

type AccountActionCountType int

const (
	AccountActionCountErc20 AccountActionCountType = iota
	AccountActionCountErc721
	AccountActionCountAction
)

func (self AccountActionCountType) String() string {
	switch self {
	case AccountActionCountErc20:
		return "erc20_count"
	case AccountActionCountErc721:
		return "erc721_count"
	case AccountActionCountAction:
		return "action_count"
	}
	return ""
}

type AccountActionCount struct {
	ID          uint64 `gorm:"primary_key;" sql:"type:bigint"`
	Address     string `gorm:"size:42;not null;default:'';uniqueIndex"`
	Erc20Count  uint64 `gorm:"type:int8;not null;default:0"` // count of erc20 token
	Erc721Count uint64 `gorm:"type:int8;not null;default:0"` // count of erc721 token
	ActionCount uint64 `gorm:"type:int8;not null;default:0"` // count of actions
}

func (m *AccountActionCount) AddCount(tx *gorm.DB, count uint64, typ AccountActionCountType) error {
	var aac AccountActionCount
	if err := tx.Model(m).Where("address = ?", m.Address).Limit(1).Find(&aac).Error; err != nil {
		return err
	}
	newCount := count
	switch typ {
	case AccountActionCountErc20:
		newCount += aac.Erc20Count
	case AccountActionCountErc721:
		newCount += aac.Erc721Count
	case AccountActionCountAction:
		newCount += aac.ActionCount
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address"}},
		DoUpdates: clause.Assignments(map[string]interface{}{typ.String(): newCount}),
	}).Create(m).Error
}
