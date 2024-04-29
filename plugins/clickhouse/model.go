package main

import (
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/shopspring/decimal"
)

type Block struct {
	BlockHeight     uint64
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

type Action struct {
	BlockHeight        uint64
	ActionHash         string
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

type Log struct {
	BlockHeight     uint64
	ActionHash      string
	ContractAddress string
	Topic0          string
	Topic1          string
	Topic2          string
	Topic3          string
	Data            []byte
	Index           uint
	TxIndex         uint
	Timestamp       time.Time
}

type TransactionLog struct {
	BlockHeight uint64
	ActionHash  string
	Type        string
	Internal    bool
	Amount      decimal.Decimal
	Sender      string
	Recipient   string
	Timestamp   time.Time
}

type AccountIncome struct {
	BlockHeight   uint64
	Address       string
	InFlow        decimal.Decimal
	InNumActions  int
	OutFlow       decimal.Decimal
	OutNumActions int
	Timestamp     time.Time
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
	if height == 0 {
		err = chDB.Migrator().DropTable(dst...)
		if err != nil {
			return 0, err
		}
		return 0, chDB.Migrator().CreateTable(dst...)
	}
	return height, nil
}
