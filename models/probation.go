package models

import (
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
)

type Probation struct {
	ID            uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight   uint64 `gorm:"not null;unsigned;index" sql:"type:bigint"`
	EpochNumber   uint64 `gorm:"not null;unsigned;index" sql:"type:bigint"`
	Address       string `gorm:"size:42;not null;default:'';"`
	IntensityRate uint32 `gorm:"type:int;not null;default:0;"`
	Count         uint64 `gorm:"not null;" sql:"type:bigint"`
}

func (Probation) TableName() string {
	return "probation"
}

// GetCountByEpochNum returns the count of a probation candidate.
func (m Probation) GetCountByEpochNum(epochNum uint64) (uint64, error) {
	var count int64
	err := db.DB().Model(m).Where("epoch_number = ?", epochNum).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return uint64(count), nil
}

// GetProbationListByEpoch returns the probation list of an epoch.
func GetProbationListByEpoch(epochNum uint64) (*iotextypes.ProbationCandidateList, error) {
	probationList := &iotextypes.ProbationCandidateList{}
	var probations []Probation
	if err := db.DB().Find(&probations, "epoch_number = ?", epochNum).Error; err != nil {
		return nil, errors.Wrapf(err, "failed to get probation list in epoch %d", epochNum)
	}
	for _, p := range probations {
		probationList.IntensityRate = p.IntensityRate
		probationList.ProbationList = append(probationList.ProbationList, &iotextypes.ProbationCandidateList_Info{
			Address: p.Address,
			Count:   uint32(p.Count),
		})
	}
	return probationList, nil
}
