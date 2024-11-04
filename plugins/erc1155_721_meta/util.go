package main

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
)

func safeHexDecode(data string) []byte {
	if data == "" {
		panic("data is empty")
	}
	data = strings.TrimPrefix(data, "0x")
	decoded, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}
	return decoded
}
func checkTopics(topics, data string) bool {
	if !strings.Contains(topics, TransferBatchString) && !strings.Contains(topics, TransferSingleString) && !strings.Contains(topics, URIString) && !strings.Contains(topics, ApprovalForAllString) {
		return false
	}
	return true
}

func isErc1155(addr, topics, data string) bool {
	if !checkTopics(topics, data) {
		return false
	}
	if _, ok := nonErc1155Contract[addr]; ok {
		return false
	}
	if _, ok := erc1155Contract[addr]; ok {
		return true
	}

	ret := readContract(addr, BalanceOf, false)
	if !ret {
		nonErc1155Contract[addr] = struct{}{}
		return false
	}

	ret = readContract(addr, BalanceOfBatch, false)
	if !ret {
		nonErc1155Contract[addr] = struct{}{}
		return false
	}
	ret = readContract(addr, SetApprovalForAll, true)
	if !ret {
		nonErc1155Contract[addr] = struct{}{}
		return false
	}
	ret = readContract(addr, IsApprovedForAll, false)
	if !ret {
		nonErc1155Contract[addr] = struct{}{}
		return false
	}
	ret = readContract(addr, SafeTransferFrom, true)
	if !ret {
		nonErc1155Contract[addr] = struct{}{}
		return false
	}
	ret = readContract(addr, SafeBatchTransferFrom, true)
	if !ret {
		nonErc1155Contract[addr] = struct{}{}
		return false
	}
	erc1155Contract[addr] = struct{}{}
	return true
}

func readContract(addr string, callData []byte, noData bool) bool {
	cli := kernel.ChainClient()
	execution, err := action.NewExecution(addr, nonce, transferAmount, gasLimit, gasPrice, callData)
	if err != nil {
		return false
	}
	request := &iotexapi.ReadContractRequest{
		Execution:     execution.Proto(),
		CallerAddress: callerAddress,
	}

	res, err := cli.ReadContract(context.Background(), request)
	if err != nil {
		return false
	}
	if res.Receipt.Status == successStatus || res.Receipt.Status == revertStatus {
		if noData {
			return true
		}
		if res.Data != "" {
			return true
		}
	}
	return false
}
