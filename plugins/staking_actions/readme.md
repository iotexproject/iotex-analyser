---
plugin: staking_action
---

## Introduction

Store staking action infomation to database, It is used to query the reward at the historical height of the account.



## Model

```
type StakingAction struct {
	ID           uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight  uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	BucketID     uint64          `gorm:"unsigned;index"`
	OwnerAddress string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Candidate    string          `gorm:"size:42;not null;default:'';index:,length:9"`
	ForwardTo    string          `gorm:"size:42;not null;default:'';"` //store Governace Forward To Address
	Amount       decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	ActType      string          `gorm:"size:42;not null;default:'';index"`
	Sender       string          `gorm:"size:42;not null;default:'';index:,length:9"`
	ActHash      string
	AutoStake    bool
	Duration     uint32
}
```