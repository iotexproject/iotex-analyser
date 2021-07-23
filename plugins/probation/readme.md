---
plugin: probation
---

## Introduction

Calculate the number of votes for candidate

## Model

```
type Probation struct {
	ID            uint64 `gorm:"primary_key;" sql:"type:bigint"`
	EpochNumber   uint64 `gorm:"not null;unsigned;index" sql:"type:bigint"`
	Address       string `gorm:"size:42;not null;default:'';"`
	IntensityRate uint32 `gorm:"type:int;not null;default:0;"`
	Count         uint64 `gorm:"not null;" sql:"type:bigint"`
}

```