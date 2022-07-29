package models

type BlockSupply struct {
	BlockHeight            uint64 `gorm:"primary_key;" sql:"type:bigint;"`
	TotalSupply            string
	TotalCirculatingSupply string
}

func (BlockSupply) TableName() string {
	return "block_supply"
}
