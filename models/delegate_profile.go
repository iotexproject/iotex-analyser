package models

import (
	"github.com/shopspring/decimal"
)

type DelegateProfileUpdated struct {
	ID          uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash  string          `gorm:"size:64;not null;index:,length:9"`
	Delegate    string          `gorm:"size:42;not null;default:'';index:,length:9;"`
	Name        string          `gorm:"size:42;not null;default:'';index:,length:9;"`
	Value       decimal.Decimal `gorm:"type:decimal(128,0);not null;default:0;"`
}

func (DelegateProfileUpdated) TableName() string {
	return "delegate_profile_updated"
}
