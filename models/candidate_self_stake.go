package models

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
