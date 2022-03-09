package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/test/identityset"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
)

const (
	//18160ddd -> totalSupply()
	totalSupplyString = "18160ddd"
	//70a08231 -> balanceOf(address)
	balanceOfString = "70a08231000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc"
	//dd62ed3e -> allowance(address,address)
	allowanceString = "dd62ed3e000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc"
	//095ea7b3 -> approve(address,uint256)
	approveString = "095ea7b3000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc0000000000000000000000000000000000000000000000000000000000000001"

	// transferSha3 is sha3 of xrc20's transfer event,keccak('Transfer(address,address,uint256)')
	transferSha3 = "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	//special for wiotx
	//Deposit(address,uint256)
	depositSha3 = "e1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c"
	//Withdrawal(address,uint256)
	withdrawalSha3 = "7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b65"

	topicsPlusDataLen = 256
	sha3Len           = 64
	contractParamsLen = 64
	addressLen        = 40
	successStatus     = uint64(1)
	revertStatus      = uint64(106)
)

var (
	totalSupply, _   = hex.DecodeString(totalSupplyString)
	balanceOf, _     = hex.DecodeString(balanceOfString)
	allowance, _     = hex.DecodeString(allowanceString)
	approve, _       = hex.DecodeString(approveString)
	erc20Contract    = make(map[string]struct{})
	nonErc20Contract = make(map[string]struct{})
	nonce            = uint64(1)
	transferAmount   = big.NewInt(0)
	gasLimit         = uint64(100000)
	gasPrice         = big.NewInt(10000000)
	callerAddress    = identityset.Address(30).String()
)

func checkTopics(topics, data string) bool {
	if topics == "" || len(topics) > 64*3 || len(data) > 64*3 {
		return false
	}
	if !strings.Contains(topics, transferSha3) && !strings.Contains(topics, depositSha3) && !strings.Contains(topics, withdrawalSha3) {
		return false
	}
	return true
}

func isErc20(addr, topics, data string) bool {
	if !checkTopics(topics, data) {
		return false
	}
	if _, ok := nonErc20Contract[addr]; ok {
		return false
	}
	if _, ok := erc20Contract[addr]; ok {
		return true
	}

	ret := readContract(addr, totalSupply)
	if !ret {
		nonErc20Contract[addr] = struct{}{}
		return false
	}

	ret = readContract(addr, balanceOf)
	if !ret {
		nonErc20Contract[addr] = struct{}{}
		return false
	}
	ret = readContract(addr, allowance)
	if !ret {
		nonErc20Contract[addr] = struct{}{}
		return false
	}
	ret = readContract(addr, approve)
	if !ret {
		nonErc20Contract[addr] = struct{}{}
		return false
	}
	erc20Contract[addr] = struct{}{}
	return true
}

func readContract(addr string, callData []byte) bool {
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
	if (res.Receipt.Status == successStatus || res.Receipt.Status == revertStatus) && res.Data != "" {
		return true
	}
	return false
}
