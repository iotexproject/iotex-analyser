---
plugin: vote_bucket
---

Save epoch VoteBucketList to db

## Introduction


## Model

```
type VoteBucketList struct {
	EpochNumber uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BucketList  []byte
}
```