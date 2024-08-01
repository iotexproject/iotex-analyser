package main

import (
	"time"

	"github.com/shopspring/decimal"
)

/*
CREATE TABLE erc20_transfers
(

	`block_height` UInt64,
	`action_hash` String,
	`contract_address` String,
	`amount` String,
	`sender` String,
	`recipient` String,
	`timestamp` DateTime64(3)

) ENGINE = ReplacingMergeTree PARTITION BY toYYYYMM(timestamp) ORDER BY (block_height, action_hash, contract_address, amount, sender, recipient, timestamp)
*/
type Erc20Transfer struct {
	BlockHeight     uint64          
	ActionHash      string          
	ContractAddress string          
	Amount          decimal.Decimal 
	Sender          string          
	Recipient       string          
	Timestamp       time.Time       
}

func (Erc20Transfer) TableName() string {
	return "erc20_transfers"
}

/*
CREATE TABLE erc20_approvals
(

	`block_height` UInt64,
	`action_hash` String,
	`contract_address` String,
	`amount` String,
	`owner` String,
	`spender` String,
	`timestamp` DateTime64(3)

) ENGINE = ReplacingMergeTree PARTITION BY toYYYYMM(timestamp) ORDER BY (block_height, action_hash, contract_address, amount, owner, spender, timestamp)
*/
type Erc20Approval struct {
	BlockHeight     uint64          
	ActionHash      string          
	ContractAddress string          
	Amount          decimal.Decimal 
	Owner           string          
	Spender         string          
	Timestamp       time.Time       
}

func (Erc20Approval) TableName() string {
	return "erc20_approvals"
}

/*
CREATE TABLE erc20_holders
(
	`contract_address` String,
	`holder` String,
	`timestamp` DateTime64(3)
) ENGINE = ReplacingMergeTree ORDER BY (contract_address, holder)
*/
type Erc20Holder struct {
	ContractAddress string
	Holder          string
	Timestamp       time.Time      
}

func (Erc20Holder) TableName() string {
	return "erc20_holders"
}