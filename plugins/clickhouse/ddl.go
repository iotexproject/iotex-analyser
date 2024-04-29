package main

const (
	blockDDL = `
    CREATE TABLE IF NOT EXISTS blocks (
		block_height UInt64,
		block_hash String,
		prev_block_hash String,
		tx_root String,
		producer_address String,
		gas_consumed UInt64,
		num_actions Int32,
		version UInt32,
		block_reward UInt256,
		epoch_reward UInt256,
		foundation_bonus UInt256,
		epoch_number UInt64,
		timestamp DateTime
	) ENGINE = MergeTree()
	PARTITION BY toYYYYMM(timestamp)
	ORDER BY block_height;`
)
