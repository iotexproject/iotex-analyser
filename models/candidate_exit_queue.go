package models

type CandidateExitQueue struct {
	ID                uint64 `gorm:"primary_key" sql:"type:bigint"`
	CandidateName     string `gorm:"size:42;not null;default:'';index"`
	CandidateIdentity string `gorm:"size:42;not null;default:'';index"`
	Status            string `gorm:"size:30;not null;default:''"`
	RequestHeight     uint64
	RequestHash       string `gorm:"size:64"`
	ScheduleHeight    uint64
	ScheduleHash      string `gorm:"size:64"`
	ConfirmHeight     uint64
	ConfirmHash       string `gorm:"size:64"`
	ScheduledAt       uint64
}

func (CandidateExitQueue) TableName() string {
	return "candidate_exit_queue"
}
