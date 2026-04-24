package models

import (
	"github.com/iotexproject/iotex-analyser/db"
)

type AccountMeta struct {
	ID               uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight      uint64 `gorm:"not null;unsigned;index" sql:"type:bigint"`
	Address          string `gorm:"size:42;not null;default:'';uniqueIndex"`
	IsContract       bool   `gorm:"type:bool;not null;default:false"`
	ContractByteCode []byte
}

func (AccountMeta) TableName() string {
	return "account_meta"
}

func (m *AccountMeta) ByAddress(addr string) error {
	return db.DB().Model(m).Where("address = ?", addr).Take(&m).Error
}

func LoadAccountContractFlags(addrs []string) (map[string]bool, error) {
	flags := make(map[string]bool, len(addrs))
	if len(addrs) == 0 {
		return flags, nil
	}
	var rows []AccountMeta
	if err := db.DB().Model(&AccountMeta{}).Select("address", "is_contract").Where("address IN ?", addrs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		flags[row.Address] = row.IsContract
	}
	return flags, nil
}

type AccountActionCountType int

const (
	AccountActionCountErc20 AccountActionCountType = iota
	AccountActionCountErc721
	AccountActionCountAction
	AccountActionCountInner
)

func (self AccountActionCountType) String() string {
	switch self {
	case AccountActionCountErc20:
		return "erc20_count"
	case AccountActionCountErc721:
		return "erc721_count"
	case AccountActionCountAction:
		return "action_count"
	case AccountActionCountInner:
		return "inner_count"
	}
	return ""
}
