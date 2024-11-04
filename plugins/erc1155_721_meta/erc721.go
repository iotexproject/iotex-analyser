package main

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
)

const (
	//313ce567 -> decimal()
	decimalString = "313ce567"
	//18160ddd -> totalSupply()
	totalSupplyString = "18160ddd"
	//70a08231 -> balanceOf(address)
	balanceOfString = "70a08231000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc"
	//095ea7b3 -> approve(address,uint256)
	approveString = "095ea7b3000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc0000000000000000000000000000000000000000000000000000000000000001"
	//6352211e -> ownerOf(uint256)
	ownerOfString = "6352211e000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc"

	//a22cb465 -> setApprovalForAll(address,bool)
	approvalForAllString = "a22cb465000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc0000000000000000000000000000000000000000000000000000000000000000"
	// TransferString is sha3 of xrc20's transfer event,keccak('Transfer(address,address,uint256)')
	TransferString = "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	//Approve(address,address,uint256)
	ApproveString = "8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"
)

var (
	ERC721ABI = `[{"inputs":[{"internalType":"string","name":"name_","type":"string"},{"internalType":"string","name":"symbol_","type":"string"}],"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"owner","type":"address"},{"indexed":true,"internalType":"address","name":"approved","type":"address"},{"indexed":true,"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"owner","type":"address"},{"indexed":true,"internalType":"address","name":"operator","type":"address"},{"indexed":false,"internalType":"bool","name":"approved","type":"bool"}],"name":"ApprovalForAll","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":true,"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"Transfer","type":"event"},{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"approve","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"getApproved","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"operator","type":"address"}],"name":"isApprovedForAll","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"ownerOf","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"safeTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"},{"internalType":"bytes","name":"data","type":"bytes"}],"name":"safeTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"operator","type":"address"},{"internalType":"bool","name":"approved","type":"bool"}],"name":"setApprovalForAll","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"bytes4","name":"interfaceId","type":"bytes4"}],"name":"supportsInterface","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"tokenURI","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"transferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"}]`

	decimals, _       = hex.DecodeString(decimalString)
	totalSupply, _    = hex.DecodeString(totalSupplyString)
	balanceOf, _      = hex.DecodeString(balanceOfString)
	approve, _        = hex.DecodeString(approveString)
	ownerOf, _        = hex.DecodeString(ownerOfString)
	erc721Contract    = make(map[string]bool)
	nonErc721Contract = make(map[string]bool)
	cachedContract    = make(map[string]struct{})
)

func isHandled(addr string) bool {
	if _, ok := cachedContract[addr]; ok {
		return true
	}
	var count int64
	m := &models.Erc1155721Meta{}
	err := db.DB().Model(m).Where("contract_address = ?", addr).Count(&count).Error
	if err != nil {
		panic(err)
	}
	return count > 0
}

func check721Topics(topics, data string) bool {
	if topics == "" || ((len(topics) != 256 && len(data) != 0) &&
		len(topics) != 192) {
		return false
	}
	if !strings.Contains(topics, TransferString) &&
		!strings.Contains(topics, ApproveString) &&
		!strings.Contains(topics, ApprovalForAllString) {
		return false
	}
	return true
}

func isErc721(addr, topics, data string) bool {
	if !check721Topics(topics, data) {
		return false
	}
	if _, ok := nonErc721Contract[addr]; ok {
		return false
	}
	if _, ok := erc721Contract[addr]; ok {
		return true
	}

	ret := readContract(addr, balanceOf, false)
	if !ret {
		nonErc721Contract[addr] = true
		return false
	}
	ret = readContract(addr, approve, false)
	if !ret {
		nonErc721Contract[addr] = true
		return false
	}
	ret = readContract(addr, ownerOf, false)
	if !ret {
		nonErc721Contract[addr] = true
		return false
	}
	ret = readContract(addr, decimals, false)
	if ret {
		nonErc721Contract[addr] = true
		return false
	}
	erc721Contract[addr] = true
	return true
}

func readERC721URI(addr string, tokenID *big.Int) (string, error) {
	cli := kernel.ChainClient()
	callData, _ := erc721ABI.Pack("tokenURI", tokenID)
	execution, err := action.NewExecution(addr, nonce, transferAmount, gasLimit, gasPrice, callData)
	if err != nil {
		return "", err
	}
	request := &iotexapi.ReadContractRequest{
		Execution:     execution.Proto(),
		CallerAddress: callerAddress,
	}

	res, err := cli.ReadContract(context.Background(), request)
	if err != nil {
		return "", err
	}
	if res.Receipt.Status == successStatus {
		if len(res.Data) == 0 {
			return "", nil
		}
		data, err := hex.DecodeString(res.Data)
		if err != nil {
			return "", err
		}
		unPack, err := erc721ABI.Unpack("tokenURI", data)
		if err != nil {
			//if can't unpack, return empty string
			return "", nil
		}
		return unPack[0].(string), nil
	}
	return "", errors.New("read erc721 contract failed")
}
