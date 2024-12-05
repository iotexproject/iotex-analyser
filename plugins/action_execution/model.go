package main

var ActionExecutionDDL = `CREATE TABLE IF NOT EXISTS action_execution
(
    block_height UInt64,
    action_hash FixedString(64),
    contract FixedString(41),
    receipt_contract_address FixedString(41),
    data String
)
ENGINE = ReplacingMergeTree()
PRIMARY KEY (action_hash)
ORDER BY (action_hash)`

type ActionExecution struct {
	BlockHeight            uint64 `ch:"block_height"`
	ActionHash             string `ch:"action_hash"`
	Contract               string `ch:"contract"`
	ReceiptContractAddress string `ch:"receipt_contract_address"`
	Data                   string `ch:"data"`
}

func (ActionExecution) TableName() string {
	return "action_execution"
}
