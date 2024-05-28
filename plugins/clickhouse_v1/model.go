package main

import (
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/shopspring/decimal"
)

/*
CREATE TABLE blocks
(

	`block_height` UInt64,
	`block_hash` String,
	`version` UInt32,
	`prev_block_hash` String,
	`tx_root` String,
	`gas_consumed` UInt64,
	`producer_address` String,
	`num_actions` Int64,
	`block_reward` String,
	`epoch_reward` String,
	`foundation_bonus` String,
	`epoch_number` UInt64,
	`timestamp` DateTime64(3)

) ENGINE = ReplacingMergeTree PARTITION BY toYYYYMM(timestamp) ORDER BY block_height
*/
type BlockV1 struct {
	BlockHeight     uint64 // order by
	BlockHash       string
	Version         uint32
	PrevBlockHash   string
	TxRoot          string
	GasConsumed     uint64
	ProducerAddress string
	NumActions      int
	BlockReward     decimal.Decimal
	EpochReward     decimal.Decimal
	FoundationBonus decimal.Decimal
	EpochNumber     uint64
	Timestamp       time.Time
}

func (BlockV1) TableName() string {
	return "blocks"
}

/*
CREATE TABLE actions
(

	`block_height` UInt64,
	`action_hash` String,
	`action_type` String,
	`sender` String,
	`recipient` String,
	`gas_price` String,
	`gas_limit` UInt64,
	`nonce` UInt64,
	`amount` String,
	`gas_consumed` UInt64,
	`contract_address` String,
	`status` UInt64,
	`execution_revert_msg` String,
	`payload` String,
	`timestamp` DateTime64(3),
	`chain_id` UInt32,
	`encoding` UInt32,
	`version` UInt32

) ENGINE = ReplacingMergeTree PARTITION BY toYYYYMM(timestamp) ORDER BY (block_height, action_hash)
*/
type ActionV1 struct {
	BlockHeight        uint64 // order by
	ActionHash         string // order by
	ActionType         string `ch:",lc"`
	Sender             string
	Recipient          string
	GasPrice           decimal.Decimal
	GasLimit           uint64
	Nonce              uint64
	Amount             decimal.Decimal
	GasConsumed        uint64
	ContractAddress    string
	Status             uint64
	ExecutionRevertMsg string
	Payload            []byte
	Timestamp          time.Time
	ChainID            uint32
	Encoding           uint32
	Version            uint32
}

func (ActionV1) TableName() string {
	return "actions"
}

/*
CREATE TABLE logs
(

	`block_height` UInt64,
	`action_hash` String,
	`contract_address` String,
	`topic0` String,
	`topic1` String,
	`topic2` String,
	`topic3` String,
	`data` String,
	`index` UInt64,
	`tx_index` UInt64,
	`timestamp` DateTime64(3)

) ENGINE = ReplacingMergeTree PARTITION BY toYYYYMM(timestamp) ORDER BY (block_height, action_hash, index, tx_index)
*/
type LogV1 struct {
	BlockHeight     uint64 // order by
	ActionHash      string // order by
	ContractAddress string
	Topic0          string
	Topic1          string
	Topic2          string
	Topic3          string
	Data            []byte
	Index           uint // order by
	TxIndex         uint // order by
	Timestamp       time.Time
}

func (LogV1) TableName() string {
	return "logs"
}

/*
CREATE TABLE transaction_logs
(

	`block_height` UInt64,
	`action_hash` String,
	`type` String,
	`internal` UInt8,
	`amount` String,
	`sender` String,
	`recipient` String,
	`timestamp` DateTime64(3)

) ENGINE = ReplacingMergeTree PARTITION BY toYYYYMM(timestamp) ORDER BY (block_height, action_hash, type, internal, amount, sender, recipient, timestamp)
*/
type TransactionLogV1 struct {
	BlockHeight uint64
	ActionHash  string
	Type        string
	Internal    bool
	Amount      decimal.Decimal
	Sender      string
	Recipient   string
	Timestamp   time.Time
}

func (TransactionLogV1) TableName() string {
	return "transaction_logs"
}

/*
CREATE TABLE account_incomes
(

	`block_height` UInt64,
	`address` String,
	`in_flow` String,
	`in_num_actions` Int64,
	`out_flow` String,
	`out_num_actions` Int64,
	`timestamp` DateTime64(3)

) ENGINE = ReplacingMergeTree PARTITION BY toYYYYMM(timestamp) ORDER BY (block_height, address, in_flow, in_num_actions, out_flow, out_num_actions, timestamp)
*/
type AccountIncomeV1 struct {
	BlockHeight   uint64
	Address       string
	InFlow        decimal.Decimal
	InNumActions  int
	OutFlow       decimal.Decimal
	OutNumActions int
	Timestamp     time.Time
}

func (AccountIncomeV1) TableName() string {
	return "account_incomes"
}

func isContractAddress(addr string) bool {
	m := &models.AccountMeta{}
	if err := m.ByAddress(addr); err != nil {
		return false
	}
	return m.IsContract
}

func AutoMigrate(index string, dst ...interface{}) (uint64, error) {
	height, err := db.GetIndexHeight(index)
	if err != nil {
		return 0, err
	}
	// TODO
	//if height == 0 {
	//	err = chDB.Migrator().DropTable(dst...)
	//	if err != nil {
	//		return 0, err
	//	}
	//	return 0, chDB.Migrator().CreateTable(dst...)
	//}
	return height, nil
}
