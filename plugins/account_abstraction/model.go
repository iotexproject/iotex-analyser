package main

type AccountAbstractionAccountDeployed struct {
	ID          uint64 `gorm:"primary_key" sql:"type:bigint"`
	BlockHeight uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash  string `gorm:"size:64;not null;index:,length:9"`
	UserOpHash  string `gorm:"size:64;not null;"`
	Sender      string `gorm:"size:42;not null;default:'';"`
	Factory     string `gorm:"size:42;not null;default:'';index:,length:9"`
	Paymaster   string `gorm:"size:42;not null;default:'';index:,length:9"`
}

func (AccountAbstractionAccountDeployed) TableName() string {
	return "account_abstraction_account_deployed"
}
