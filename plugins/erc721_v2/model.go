package main

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	versionSuffix = "_v" + strings.ReplaceAll(VERSION, ".", "_")
)

var Erc721TransferDDL = `CREATE TABLE IF NOT EXISTS erc721_transfers_v2_2_3
(
    block_height UInt64,
    action_hash String,
    contract_address String,
    token_id String,
    sender String,
    recipient String,
    timestamp DateTime64(6)
)
ENGINE = ReplacingMergeTree()
PRIMARY KEY action_hash
ORDER BY action_hash`

type Erc721Transfer struct {
	BlockHeight     uint64    `ch:"block_height"`
	ActionHash      string    `ch:"action_hash"`
	ContractAddress string    `ch:"contract_address"`
	TokenId         string    `ch:"token_id"`
	Sender          string    `ch:"sender"`
	Recipient       string    `ch:"recipient"`
	Timestamp       time.Time `ch:"timestamp"`
}

func (Erc721Transfer) TableName() string {
	return "erc721_transfers" + versionSuffix
}

var Erc721ApprovalDDL = `CREATE TABLE IF NOT EXISTS erc721_approvals_v2_2_3
(
    block_height UInt64,
    action_hash String,
    contract_address String,
    owner String,
    approved String,
    token_id String,
    timestamp DateTime64(6)
)
ENGINE = ReplacingMergeTree()
PRIMARY KEY action_hash
ORDER BY action_hash`

type Erc721Approval struct {
	BlockHeight     uint64    `ch:"block_height"`
	ActionHash      string    `ch:"action_hash"`
	ContractAddress string    `ch:"contract_address"`
	Owner           string    `ch:"owner"`
	Approved        string    `ch:"approved"`
	TokenId         string    `ch:"token_id"`
	Timestamp       time.Time `ch:"timestamp"`
}

func (Erc721Approval) TableName() string {
	return "erc721_approvals" + versionSuffix
}

var Erc721HolderDDL = `CREATE TABLE IF NOT EXISTS erc721_holders_v2_2_3
(
    contract_address String,
    holder String,
)
ENGINE = ReplacingMergeTree()
PRIMARY KEY (contract_address, holder)
ORDER BY (contract_address, holder)`

type Erc721Holder struct {
	ContractAddress string `ch:"contract_address"`
	Holder          string `ch:"holder"`
}

func (Erc721Holder) TableName() string {
	return "erc721_holders" + versionSuffix
}

var Erc721ApprovalForAllDDL = `CREATE TABLE IF NOT EXISTS erc721_approval_for_alls_v2_2_3
(
    block_height UInt64,
    action_hash String,
    contract_address String,
    owner String,
    operator String,
    approved Bool,
    timestamp DateTime64(6)
)
ENGINE = ReplacingMergeTree()
PRIMARY KEY action_hash
ORDER BY action_hash`

type Erc721ApprovalForAll struct {
	BlockHeight     uint64    `ch:"block_height"`
	ActionHash      string    `ch:"action_hash"`
	ContractAddress string    `ch:"contract_address"`
	Owner           string    `ch:"owner"`
	Operator        string    `ch:"operator"`
	Approved        bool      `ch:"approved"`
	Timestamp       time.Time `ch:"timestamp"`
}

func (Erc721ApprovalForAll) TableName() string {
	return "erc721_approval_for_alls" + versionSuffix
}

func (b tokenPlugin) migrateTable(ctx context.Context) error {
	if err := b.conn.Exec(ctx, Erc721TransferDDL); err != nil {
		return errors.Wrap(err, "failed to create clickhouse erc721_transfers_v2_2_3 table")
	}
	if err := b.conn.Exec(ctx, Erc721ApprovalDDL); err != nil {
		return errors.Wrap(err, "failed to create clickhouse erc721_approvals_v2_2_3 table")
	}
	if err := b.conn.Exec(ctx, Erc721HolderDDL); err != nil {
		return errors.Wrap(err, "failed to create clickhouse erc721_holders_v2_2_3 table")
	}
	err := b.conn.Exec(ctx, Erc721ApprovalForAllDDL)
	return errors.Wrap(err, "failed to create clickhouse erc721_approval_for_alls_v2_2_3 table")
}
