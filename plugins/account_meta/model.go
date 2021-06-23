package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
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

func appendIfMissing(slice []string, s string) []string {
	for _, element := range slice {
		if element == s {
			return slice
		}
	}
	return append(slice, s)
}

func accountMeta(addr string) (*iotextypes.AccountMeta, error) {

	ctx := context.Background()
	request := iotexapi.GetAccountRequest{Address: addr}
	cli := kernel.ChainClient()
	resp, err := cli.GetAccount(ctx, &request)
	if err != nil {
		return nil, err
	}

	return resp.AccountMeta, nil
}
