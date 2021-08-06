package models

import (
	"github.com/iotexproject/iotex-analyser/db"
)

type Probation struct {
	ID            uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight   uint64 `gorm:"not null;unsigned;index" sql:"type:bigint"`
	EpochNumber   uint64 `gorm:"not null;unsigned;index" sql:"type:bigint"`
	Address       string `gorm:"size:42;not null;default:'';"`
	IntensityRate uint32 `gorm:"type:int;not null;default:0;"`
	Count         uint64 `gorm:"not null;" sql:"type:bigint"`
}

func (Probation) TableName() string {
	return "probation"
}

func (m Probation) GetCountByEpochNum(epochNum uint64) (uint64, error) {
	var count int64
	err := db.DB().Model(m).Where("epoch_number = ?", epochNum).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return uint64(count), nil
}
