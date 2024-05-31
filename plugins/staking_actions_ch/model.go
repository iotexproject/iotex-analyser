package main

import (
	"time"

	"github.com/shopspring/decimal"
)

/*
CREATE TABLE testnet.staking_actions
(

	`block_height` UInt64,
	`index` Int64,
	`bucket_id` UInt64,
	`owner_address` String,
	`candidate` String,
	`amount` String,
	`act_type` String,
	`sender` String,
	`act_hash` String,
	`auto_stake` UInt8,
	`duration` UInt32,
	`timestamp` DateTime64(3)

) ENGINE = ReplacingMergeTree PARTITION BY toYYYYMM(timestamp) ORDER BY (block_height, index, bucket_id, owner_address, candidate, amount, act_type, sender, act_hash, auto_stake, duration, timestamp)
*/
type StakingActions struct {
	BlockHeight  uint64
	Index        int
	BucketID     uint64
	OwnerAddress string
	Candidate    string
	Amount       decimal.Decimal
	ActType      string
	Sender       string
	ActHash      string
	AutoStake    bool
	Duration     uint32
	Timestamp    time.Time
}

func (StakingActions) TableName() string {
	return "staking_actions"
}
