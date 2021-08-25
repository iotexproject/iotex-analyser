package main

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/iotex-address/address"
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
	//6352211e -> ownerOf(uint256)
	ownerOfString = "6352211e000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc"
	// transferSha3 is sha3 of xrc20's transfer event,keccak('Transfer(address,address,uint256)')
	transferSha3 = "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	topicsPlusDataLen = 256
	sha3Len           = 64
	contractParamsLen = 64
	addressLen        = 40
	successStatus     = uint64(1)
	revertStatus      = uint64(106)
)

var (
	totalSupply, _    = hex.DecodeString(totalSupplyString)
	balanceOf, _      = hex.DecodeString(balanceOfString)
	allowance, _      = hex.DecodeString(allowanceString)
	approve, _        = hex.DecodeString(approveString)
	ownerOf, _        = hex.DecodeString(ownerOfString)
	erc721Contract    = make(map[string]bool)
	nonErc721Contract = make(map[string]bool)
	nonce             = uint64(1)
	transferAmount    = big.NewInt(0)
	gasLimit          = uint64(100000)
	gasPrice          = big.NewInt(10000000)
	callerAddress     = identityset.Address(30).String()
)

func checkTopics(topics, data string) bool {
	if topics == "" || len(topics) != 256 || len(data) != 0 {
		return false
	}
	if !strings.Contains(topics, transferSha3) {
		return false
	}
	return true
}

func isErc721(addr, topics, data string) bool {
	if !checkTopics(topics, data) {
		return false
	}
	if _, ok := nonErc721Contract[addr]; ok {
		return false
	}
	if _, ok := erc721Contract[addr]; ok {
		return true
	}

	ret := readContract(addr, totalSupply)
	if !ret {
		nonErc721Contract[addr] = true
		return false
	}

	ret = readContract(addr, balanceOf)
	if !ret {
		nonErc721Contract[addr] = true
		return false
	}
	ret = readContract(addr, approve)
	if !ret {
		nonErc721Contract[addr] = true
		return false
	}
	ret = readContract(addr, ownerOf)
	if !ret {
		nonErc721Contract[addr] = true
		return false
	}
	erc721Contract[addr] = true
	return true
}

// ParseContractData parse xrc20 topics
func ParseContractData(topics, data string) (from, to string, amount *big.Int, err error) {
	// This should cover input of indexed or not indexed ,i.e., len(topics)==192 len(data)==64 or len(topics)==64 len(data)==192
	all := topics + data
	if len(all) != topicsPlusDataLen {
		err = errors.New("data's len is wrong")
		return
	}
	fromEth := all[sha3Len+contractParamsLen-addressLen : sha3Len+contractParamsLen]
	ethAddress := common.HexToAddress(fromEth)
	ioAddress, err := address.FromBytes(ethAddress.Bytes())
	if err != nil {
		return
	}
	from = ioAddress.String()

	toEth := all[sha3Len+contractParamsLen*2-addressLen : sha3Len+contractParamsLen*2]
	ethAddress = common.HexToAddress(toEth)
	ioAddress, err = address.FromBytes(ethAddress.Bytes())
	if err != nil {
		return
	}
	to = ioAddress.String()

	amount, ok := new(big.Int).SetString(all[sha3Len+contractParamsLen*2:], 16)
	if !ok {
		err = errors.New("amount convert error")
		return
	}
	return
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
