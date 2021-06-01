package main

type NodeDelegates struct {
	ID              uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64 `gorm:"unsigned" sql:"type:bigint;index"`
	ProducerAddress string `gorm:"size:42;not null;default:'';"`
	Active          bool   `gorm:"type:bool;not null;default:false"`
	ProducerName    string `gorm:"size:42;not null;default:'';"`
	Rank            int    `gorm:"type:int2;unsigned;not null;default:0"`
	Blocks          int    `gorm:"type:int4;unsigned;not null;default:0"`
	Votes           string `gorm:"size:64;not null;default:'';"`
	Probated        bool   `gorm:"type:bool;not null;default:false"`
}

func (NodeDelegates) TableName() string {
	return "node_delegates"
}
