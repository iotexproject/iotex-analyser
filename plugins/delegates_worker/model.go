package main

import "github.com/shopspring/decimal"

type NodeDelegates struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	ProducerAddress string          `gorm:"size:42;not null;default:'';"`
	Active          bool            `gorm:"type:bool;not null;default:false"`
	ProducerName    string          `gorm:"size:42;not null;default:'';"`
	Rank            int             `gorm:"type:int2;unsigned;not null;default:0"`
	Blocks          int             `gorm:"type:int4;unsigned;not null;default:0"`
	Votes           decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	Probated        bool            `gorm:"type:bool;not null;default:false"`
}

func (NodeDelegates) TableName() string {
	return "node_delegates"
}
