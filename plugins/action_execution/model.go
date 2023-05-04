package main

type ActionExecution struct {
	ID                     uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight            uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash             string `gorm:"size:64;not null;index:,length:9"`
	Contract               string `gorm:"size:42;not null;default:'';index:,length:9"`
	ReceiptContractAddress string `gorm:"size:42;not null;default:'';index:,length:9"`
	Data                   []byte `gorm:"not null;"`
}

func (ActionExecution) TableName() string {
	return "action_execution"
}
