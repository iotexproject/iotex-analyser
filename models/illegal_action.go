package models

type IllegalAction struct {
	ID          uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash  string `gorm:"size:64;not null;index:,length:9"`
	Sender      string
	Recipient   string
}

func (IllegalAction) TableName() string {
	return "illegal_actions"
}
