package main

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
	CandidateID     string          `gorm:"size:42;not null;default:'';"`
	Amount          decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	Duration        uint32          `gorm:"not null;" sql:"type:bigint"`
	ActType         string
	AutoStake       bool
	Payload         []byte
}

type Candidates []Candidate

func (Candidate) TableName() string {
	return "candidate_v1"
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

func (m *Candidate) FetchByOwnerAddressWithHeight(owner string, height uint64, candidates []*Candidate) error {
	for i := len(candidates) - 1; i >= 0; i-- {
		if candidates[i].OwnerAddress == owner && candidates[i].BlockHeight <= height {
			m = &Candidate{
				BlockHeight:     candidates[i].BlockHeight,
				Name:            candidates[i].Name,
				OperatorAddress: candidates[i].OperatorAddress,
				RewardAddress:   candidates[i].RewardAddress,
				OwnerAddress:    candidates[i].OwnerAddress,
				CandidateID:     candidates[i].CandidateID,
				Amount:          candidates[i].Amount,
				Duration:        candidates[i].Duration,
				ActType:         candidates[i].ActType,
				AutoStake:       candidates[i].AutoStake,
				Payload:         candidates[i].Payload,
			}
			return nil
		}
	}

	var err error
	db := db.DB()
	err = db.Model(m).Where("block_height <=? and owner_address = ?", height, owner).Order("block_height desc,id desc").Take(&m).Error
	return err
}

func (m *Candidate) FetchByCandidateIDWithHeight(candidateID string, height uint64, candidates []*Candidate) error {
	for i := len(candidates) - 1; i >= 0; i-- {
		if candidates[i].CandidateID == candidateID && candidates[i].BlockHeight <= height {
			m = &Candidate{
				BlockHeight:     candidates[i].BlockHeight,
				Name:            candidates[i].Name,
				OperatorAddress: candidates[i].OperatorAddress,
				RewardAddress:   candidates[i].RewardAddress,
				OwnerAddress:    candidates[i].OwnerAddress,
				CandidateID:     candidates[i].CandidateID,
				Amount:          candidates[i].Amount,
				Duration:        candidates[i].Duration,
				ActType:         candidates[i].ActType,
				AutoStake:       candidates[i].AutoStake,
				Payload:         candidates[i].Payload,
			}
			return nil
		}
	}

	var err error
	db := db.DB()
	err = db.Model(m).Where("block_height <=? and candidate_id = ?", height, candidateID).Order("block_height desc,id desc").Take(&m).Error
	return err
}

func (m *Candidate) FetchByNameWithHeight(name string, height uint64, candidates []*Candidate) error {
	for i := len(candidates) - 1; i >= 0; i-- {
		if candidates[i].Name == name && candidates[i].BlockHeight <= height {
			m = &Candidate{
				BlockHeight:     candidates[i].BlockHeight,
				Name:            candidates[i].Name,
				OperatorAddress: candidates[i].OperatorAddress,
				RewardAddress:   candidates[i].RewardAddress,
				OwnerAddress:    candidates[i].OwnerAddress,
				CandidateID:     candidates[i].CandidateID,
				Amount:          candidates[i].Amount,
				Duration:        candidates[i].Duration,
				ActType:         candidates[i].ActType,
				AutoStake:       candidates[i].AutoStake,
				Payload:         candidates[i].Payload,
			}
			return nil
		}
	}

	var err error
	db := db.DB()
	err = db.Model(m).Where("block_height <=? and name = ?", height, name).Order("block_height desc,id desc").Take(&m).Error
	return err
}
