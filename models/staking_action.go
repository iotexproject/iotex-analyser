package models

var StakingActionsDDL = `CREATE TABLE IF NOT EXISTS staking_actions
(
	block_height UInt64 NOT NULL,
	bucket_id String NOT NULL,
	owner_address FixedString(41) NOT NULL,
	candidate FixedString(41) NOT NULL,
	amount String NOT NULL,
	act_type String NOT NULL,
	sender FixedString(41) NOT NULL,
	act_hash FixedString(64) NOT NULL,
	auto_stake Bool NOT NULL,
	duration UInt32 NOT NULL,
	log_index UInt32 NOT NULL
)
ENGINE = ReplacingMergeTree()
PRIMARY KEY (bucket_id, block_height, log_index)
ORDER BY (bucket_id, block_height, log_index)`

type StakingActions struct {
	BlockHeight  uint64 `ch:"block_height"`
	BucketID     string `ch:"bucket_id"`
	OwnerAddress string `ch:"owner_address"`
	Candidate    string `ch:"candidate"`
	Amount       string `ch:"amount"`
	ActType      string `ch:"act_type"`
	Sender       string `ch:"sender"`
	ActHash      string `ch:"act_hash"`
	AutoStake    bool   `ch:"auto_stake"`
	Duration     uint32 `ch:"duration"`
	LogIndex     uint32 `ch:"log_index"`
}

func (StakingActions) TableName() string {
	return "staking_actions"
}
