---
plugin: vote_bucket
---

Save epoch CandidateList to db

## Introduction


## Model

```
type CandidateList struct {
	EpochNumber   uint64 `gorm:"primary_key;" sql:"type:bigint"`
	CandidateList []byte
}
```