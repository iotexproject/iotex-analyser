package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
)

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
