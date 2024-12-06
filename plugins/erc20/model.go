package main

import (
	"time"
)

var Erc20TransferDDL = `CREATE TABLE IF NOT EXISTS erc20_transfers
(
    block_height UInt64 NOT NULL,
    log_index UInt32 NOT NULL,
    action_hash FixedString(64) NOT NULL,
    contract_address FixedString(41) NOT NULL,
    amount String NOT NULL,
    sender FixedString(41) NOT NULL,
    recipient FixedString(41) NOT NULL,
    timestamp DateTime64(6) NOT NULL
)
ENGINE = ReplacingMergeTree()
PRIMARY KEY (block_height, log_index)
ORDER BY (block_height, log_index)`

type Erc20Transfer struct {
	BlockHeight     uint64    `ch:"block_height"`
	LogIndex        uint32    `ch:"log_index"`
	ActionHash      string    `ch:"action_hash"`
	ContractAddress string    `ch:"contract_address"`
	Amount          string    `ch:"amount"`
	Sender          string    `ch:"sender"`
	Recipient       string    `ch:"recipient"`
	Timestamp       time.Time `ch:"timestamp"`
}

func (Erc20Transfer) TableName() string {
	return "erc20_transfers"
}

var Erc20ApprovalDDL = `CREATE TABLE IF NOT EXISTS erc20_approvals
(
    block_height UInt64 NOT NULL,
    log_index UInt32 NOT NULL,
    action_hash FixedString(64) NOT NULL,
    contract_address FixedString(41) NOT NULL,
    owner FixedString(41) NOT NULL,
    spender FixedString(41) NOT NULL,
    amount String NOT NULL,
    timestamp DateTime64(6) NOT NULL
)
ENGINE = ReplacingMergeTree()
PRIMARY KEY (block_height, log_index)
ORDER BY (block_height, log_index)`

type Erc20Approval struct {
	BlockHeight     uint64    `ch:"block_height"`
	LogIndex        uint32    `ch:"log_index"`
	ActionHash      string    `ch:"action_hash"`
	ContractAddress string    `ch:"contract_address"`
	Amount          string    `ch:"amount"`
	Owner           string    `ch:"owner"`
	Spender         string    `ch:"spender"`
	Timestamp       time.Time `ch:"timestamp"`
}

func (Erc20Approval) TableName() string {
	return "erc20_approvals"
}

var Erc20HolderDDL = `CREATE TABLE IF NOT EXISTS erc20_holders
(
    contract_address FixedString(41) NOT NULL,
    holder FixedString(41) NOT NULL,
	timestamp DateTime64(6) NOT NULL
)
ENGINE = ReplacingMergeTree()
PRIMARY KEY (contract_address, holder)
ORDER BY (contract_address, holder)`

type Erc20Holder struct {
	ContractAddress string    `ch:"contract_address"`
	Holder          string    `ch:"holder"`
	Timestamp       time.Time `ch:"timestamp"`
}

func (Erc20Holder) TableName() string {
	return "erc20_holders"
}
