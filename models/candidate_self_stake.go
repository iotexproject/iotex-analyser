package models

import "gorm.io/gorm"

type CandidateSelfStake struct {
	BlockHeight uint64 `gorm:"not null;unsigned;index" sql:"type:bigint"`
	ActionHash  string `gorm:"size:64;not null;index:,length:9;uniqueIndex:idx_candidate_action"`
	TxIndex     int    `gorm:"not null;unsigned"`
	CandidateID string `gorm:"size:42;not null;index;uniqueIndex:idx_candidate_action"`
	BucketID    uint64 `gorm:"unsigned;index;type:numeric(20,0)"`
}

func (CandidateSelfStake) TableName() string {
	return "candidate_self_stake"
}

func (m *CandidateSelfStake) FetchByCandidateIDWithHeight(candidateID string, height uint64, tx *gorm.DB) error {
	var err error
	err = tx.Model(m).Where("block_height <=? and candidate_id = ?", height, candidateID).Order("block_height desc,tx_index desc").Take(&m).Error
	return err
}
