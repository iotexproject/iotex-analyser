package models

import (
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type CandidateList struct {
	EpochNumber   uint64 `gorm:"primary_key;" sql:"type:bigint"`
	CandidateList []byte
}

func (CandidateList) TableName() string {
	return "candidate_list"
}

func GetCandidateList(epochNum uint64) (*iotextypes.CandidateListV2, error) {
	candidateListAll := &iotextypes.CandidateListV2{}
	var vbl CandidateList
	if err := db.DB().Where("epoch_number = ?", epochNum).First(&vbl).Error; err != nil {
		return nil, errors.Wrapf(err, "failed to get candidate list in epoch %d", epochNum)
	}
	if err := proto.Unmarshal(vbl.CandidateList, candidateListAll); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal candidate list in epoch %d", epochNum)
	}
	return candidateListAll, nil
}
