package main

import (
	"github.com/iotexproject/iotex-analyser/db"
	"gorm.io/gorm"
)

type Block struct {
	db.BaseModel
	BlockHeight     uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHash       string `gorm:"size:64;not null;"`
	ProducerAddress string `gorm:"size:42;not null;index:,length:9"`
	NumActions      int    `gorm:"type:int2;unsigned;not null;default:0"`
	Timestamp       int64  `gorm:"type:int4;unsigned;not null;default:0"`
	Year            int    `gorm:"type:int2; unsigned;not null;default:0;index:idx_ymd"`
	Month           int    `gorm:"type:int2; unsigned;not null;default:0;index:idx_ymd"`
	Day             int    `gorm:"type:int2; unsigned;not null;default:0;index:idx_ymd"`
}

func (Block) TableName() string {
	return "block"
}

func (m *Block) Save() error {
	if err := db.DB().Create(&m).Error; err != nil {
		return err
	}
	return nil
}

func (m *Block) AfterSave(tx *gorm.DB) error {
	return db.UpdateIndexHeight(m.IndexName, m.IndexHeight)
}
