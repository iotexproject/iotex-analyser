package models

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Slash struct {
	BlockHeight     uint64          `gorm:"not null;unsigned;index;uniqueIndex:idx_block_operator" sql:"type:bigint"`
	ActionHash      string          `gorm:"size:64;not null;index:,length:9"`
	OperatorAddress string          `gorm:"size:42;index;not null;uniqueIndex:idx_block_operator"`
	CandidateID     string          `gorm:"size:42;not null;index"`
	BucketID        uint64          `gorm:"unsigned;index;type:numeric(20,0)"`
	Amount          decimal.Decimal `gorm:"type:decimal(60,0);not null"`
}

func (Slash) TableName() string {
	return "slash"
}

func FetchSlashByActionHash(hash string, tx *gorm.DB) ([]Slash, error) {
	var slashes []Slash
	if err := tx.Where("action_hash=?", hash).Find(&slashes).Error; err != nil {
		return nil, err
	}
	return slashes, nil
}
