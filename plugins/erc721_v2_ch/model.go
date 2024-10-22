package main

import (
	"time"

	"github.com/shopspring/decimal"
)

/*
CREATE TABLE erc721_transfers
(

	`block_height` UInt64,
	`action_hash` String,
	`contract_address` String,
	`token_id` String,
	`sender` String,
	`recipient` String,
	`timestamp` DateTime64(3)

) ENGINE = ReplacingMergeTree PARTITION BY toYYYYMM(timestamp) ORDER BY (block_height, action_hash, contract_address, token_id, sender, recipient, timestamp)
*/
type Erc721Transfer struct {
	BlockHeight     uint64
	ActionHash      string
	ContractAddress string
	TokenId         decimal.Decimal
	Sender          string
	Recipient       string
	Timestamp       time.Time
}

func (Erc721Transfer) TableName() string {
	return "erc721_transfers"
}

/*
CREATE TABLE erc721_approvals
(

	`block_height` UInt64,
	`action_hash` String,
	`contract_address` String,
	`owner` String,
	`approved` String,
	`token_id` String,
	`timestamp` DateTime64(3)

) ENGINE = ReplacingMergeTree PARTITION BY toYYYYMM(timestamp) ORDER BY (block_height, action_hash, contract_address, owner, approved, token_id, timestamp)
*/
type Erc721Approval struct {
	BlockHeight     uint64
	ActionHash      string
	ContractAddress string
	Owner           string
	Approved        string
	TokenId         decimal.Decimal
	Timestamp       time.Time
}

func (Erc721Approval) TableName() string {
	return "erc721_approvals"
}

/*
CREATE TABLE erc721_holders
(

	`contract_address` String,
	`holder` String,
	`timestamp` DateTime64(3)

) ENGINE = ReplacingMergeTree ORDER BY (contract_address, holder)
*/
type Erc721Holder struct {
	ContractAddress string
	Holder          string
}

func (Erc721Holder) TableName() string {
	return "erc721_holders"
}

/*
CREATE TABLE erc721_approval_for_alls
(

	`block_height` UInt64,
	`action_hash` String,
	`contract_address` String,
	`owner` String,
	`operator` String,
	`approved` Bool,
	`timestamp` DateTime64(3)

) ENGINE = ReplacingMergeTree PARTITION BY toYYYYMM(timestamp) ORDER BY (block_height, action_hash, contract_address, owner, operator, approved, timestamp)
*/
type Erc721ApprovalForAll struct {
	BlockHeight     uint64
	ActionHash      string
	ContractAddress string
	Owner           string
	Operator        string
	Approved        bool
	Timestamp       time.Time
}

func (Erc721ApprovalForAll) TableName() string {
	return "erc721_approval_for_alls"
}
