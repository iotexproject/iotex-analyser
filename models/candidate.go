package models

import (
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/shopspring/decimal"
)

type Candidate struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"not null;unsigned;index" sql:"type:bigint"`
	Name            string          `gorm:"size:42;not null;default:'';index"`
	OperatorAddress string          `gorm:"size:42;not null;default:'';"`
	RewardAddress   string          `gorm:"size:42;not null;default:'';"`
	OwnerAddress    string          `gorm:"size:42;not null;default:'';"`
	Amount          decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	Duration        uint32          `gorm:"not null;" sql:"type:bigint"`
	ActType         string
	AutoStake       bool
	Payload         []byte
}

func (Candidate) TableName() string {
	return "candidate"
}

func (m *Candidate) FetchByName(name string) (*Candidate, error) {
	var err error
	db := db.DB()
	err = db.Model(m).Where("name = ?", name).Order("block_height desc,id desc").Take(&m).Error
	if err != nil {
		return nil, err
	}
	return m, err
}
