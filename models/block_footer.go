package models

type BlockFooter struct {
	BlockHeight uint64 `gorm:"primary_key;" sql:"type:bigint;"`
	Endorser    string `gorm:"primary_key;size:42;not null;default:'';index"`
}

func (BlockFooter) TableName() string {
	return "block_footer"
}
