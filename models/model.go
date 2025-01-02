package models

import "time"

type Block struct {
	BlockHeight     uint64    `gorm:"primary_key;" sql:"type:bigint"`
	BlockHash       string    `gorm:"size:64;not null;uniqueIndex"`
	ProducerAddress string    `gorm:"size:42;not null"`
	NumActions      int       `gorm:"type:int2;unsigned;not null;default:0"`
	Timestamp       time.Time `gorm:"type:timestamp"`
	Year            int       `gorm:"type:int2;unsigned;not null;default:0"`
	Month           int       `gorm:"type:int2;unsigned;not null;default:0"`
	Day             int       `gorm:"type:int2;unsigned;not null;default:0"`
}

func (Block) TableName() string {
	return "block"
}
