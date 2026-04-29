package models

import (
	"errors"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
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

func (m *Candidate) FetchByOwnerAddressWithHeight(owner string, height uint64, tx *gorm.DB) error {
	return tx.Model(&Candidate{}).Where("block_height <= ? AND owner_address = ?", height, owner).Order("block_height DESC, id DESC").Take(m).Error
}

func (m *Candidate) FetchByOperatorAddressWithHeight(operator string, height uint64, tx *gorm.DB) error {
	return tx.Model(&Candidate{}).Where("block_height <= ? AND operator_address = ?", height, operator).Order("block_height DESC, id DESC").Take(m).Error
}

func (m *Candidate) FetchByCandidateIDWithHeight(candidateID string, height uint64, tx *gorm.DB) error {
	return tx.Model(&Candidate{}).Where("block_height <= ? AND candidate_id = ?", height, candidateID).Order("block_height DESC, id DESC").Take(m).Error
}

// FetchByOwnerOrOperatorWithHeight finds the latest candidate row whose
// `owner_address` OR `operator_address` matches the supplied address.
// CandidateUpdate / CandidateTransferOwnership are accepted by the chain when
// signed by either the owner or the (current) operator, so the plugin must
// look up the candidate via either field.
func (m *Candidate) FetchByOwnerOrOperatorWithHeight(addr string, height uint64, tx *gorm.DB) error {
	return tx.Model(&Candidate{}).
		Where("block_height <= ? AND (owner_address = ? OR operator_address = ?)", height, addr, addr).
		Order("block_height DESC, id DESC").
		Take(m).Error
}

func (m *Candidate) FetchByNameWithHeight(name string, height uint64) error {
	var err error
	db := db.DB()
	err = db.Model(m).Where("block_height <=? and name = ?", height, name).Order("block_height desc,id desc").Take(&m).Error
	return err
}

func GetAllCandidates() (Candidates, error) {
	var candidates Candidates
	db := db.DB()
	result := db.Model(&Candidate{}).Where("id in (select max(id) from candidate group by candidate_id)").Find(&candidates)
	return candidates, result.Error
}

func (m Candidates) ByOwnerAddress(addr string) (Candidate, error) {
	for _, cand := range m {
		if cand.OwnerAddress == addr {
			return cand, nil
		}
	}
	return Candidate{}, errors.New("not found: " + addr)
}

func (m Candidates) ByCandidateID(addr string) (Candidate, error) {
	for _, cand := range m {
		if cand.CandidateID == addr {
			return cand, nil
		}
	}
	return Candidate{}, errors.New("not found: " + addr)
}
