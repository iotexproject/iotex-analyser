package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
)

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

func chunkStrings(items []string, chunkSize int) [][]string {
	if chunkSize <= 0 || len(items) == 0 {
		return nil
	}
	chunks := make([][]string, 0, (len(items)+chunkSize-1)/chunkSize)
	for start := 0; start < len(items); start += chunkSize {
		end := start + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[start:end])
	}
	return chunks
}
