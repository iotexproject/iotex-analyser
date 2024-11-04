package main

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
)

var ERC1155ABI = `[{"inputs":[],"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"account","type":"address"},{"indexed":true,"internalType":"address","name":"operator","type":"address"},{"indexed":false,"internalType":"bool","name":"approved","type":"bool"}],"name":"ApprovalForAll","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"previousOwner","type":"address"},{"indexed":true,"internalType":"address","name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"account","type":"address"}],"name":"Paused","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"operator","type":"address"},{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256[]","name":"ids","type":"uint256[]"},{"indexed":false,"internalType":"uint256[]","name":"values","type":"uint256[]"}],"name":"TransferBatch","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"operator","type":"address"},{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256","name":"id","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"TransferSingle","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"string","name":"value","type":"string"},{"indexed":true,"internalType":"uint256","name":"id","type":"uint256"}],"name":"URI","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"account","type":"address"}],"name":"Unpaused","type":"event"},{"inputs":[{"internalType":"address","name":"account","type":"address"},{"internalType":"uint256","name":"id","type":"uint256"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address[]","name":"accounts","type":"address[]"},{"internalType":"uint256[]","name":"ids","type":"uint256[]"}],"name":"balanceOfBatch","outputs":[{"internalType":"uint256[]","name":"","type":"uint256[]"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"},{"internalType":"address","name":"operator","type":"address"}],"name":"isApprovedForAll","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"owner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"pause","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"paused","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"renounceOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256[]","name":"ids","type":"uint256[]"},{"internalType":"uint256[]","name":"amounts","type":"uint256[]"},{"internalType":"bytes","name":"data","type":"bytes"}],"name":"safeBatchTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"id","type":"uint256"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"bytes","name":"data","type":"bytes"}],"name":"safeTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"operator","type":"address"},{"internalType":"bool","name":"approved","type":"bool"}],"name":"setApprovalForAll","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"string","name":"newuri","type":"string"}],"name":"setURI","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"bytes4","name":"interfaceId","type":"bytes4"}],"name":"supportsInterface","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"unpause","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint256","name":"","type":"uint256"}],"name":"uri","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"}]`
var interfaceId = [4]byte{180, 90, 60, 14}

const (
	// Solidity: event TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
	TransferBatchString = "4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb"
	// Solidity: event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
	TransferSingleString = "c3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62"
	// Solidity: event URI(string value, uint256 indexed id)
	URIString = "6bb7ff708619ba0610cba295a58592e0451dee2622938c8755667688daf3529b"
	// Solidity: event ApprovalForAll(address indexed account, address indexed operator, bool approved)
	ApprovalForAllString = "17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31"
	//https://adibas03.github.io/online-ethereum-abi-encoder-decoder/#/encode
	// Solidity: function balanceOf(address account, uint256 id) view returns(uint256)
	BalanceOfString = "00fdd58e000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc0000000000000000000000000000000000000000000000000000000000000001"
	// Solidity: function balanceOfBatch(address[] accounts, uint256[] ids) view returns(uint256[])
	BalanceOfBatchString = "4e1273f4000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000001000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001"
	// Solidity: function setApprovalForAll(address operator, bool approved) returns()
	SetApprovalForAllString = "a22cb465000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc0000000000000000000000000000000000000000000000000000000000000000"
	// Solidity: function isApprovedForAll(address account, address operator) view returns(bool)
	IsApprovedForAllString = "e985e9c5000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc"
	// Solidity: function safeTransferFrom(address from, address to, uint256 id, uint256 amount, bytes data) returns()
	SafeTransferFromString = "f242432a000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc"
	// Solidity: function safeBatchTransferFrom(address from, address to, uint256[] ids, uint256[] amounts, bytes data) returns()
	SafeBatchTransferFromString = "2eb2c2d6000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc000000000000000000000000fea7d8ac16886585f1c232f13fefc3cfa26eb4cc"
	topicsPlusDataLen           = 256
	sha3Len                     = 64
	contractParamsLen           = 64
	addressLen                  = 40
	successStatus               = uint64(1)
	revertStatus                = uint64(106)
)

var (
	TransferBatch         = safeHexDecode(TransferBatchString)
	TransferSingle        = safeHexDecode(TransferSingleString)
	URI                   = safeHexDecode(URIString)
	ApprovalForAll        = safeHexDecode(ApprovalForAllString)
	BalanceOf             = safeHexDecode(BalanceOfString)
	BalanceOfBatch        = safeHexDecode(BalanceOfBatchString)
	SetApprovalForAll     = safeHexDecode(SetApprovalForAllString)
	IsApprovedForAll      = safeHexDecode(IsApprovedForAllString)
	SafeTransferFrom      = safeHexDecode(SafeTransferFromString)
	SafeBatchTransferFrom = safeHexDecode(SafeBatchTransferFromString)

	erc1155Contract    = make(map[string]struct{})
	nonErc1155Contract = make(map[string]struct{})
	nonce              = uint64(1)
	transferAmount     = big.NewInt(0)
	gasLimit           = uint64(100000)
	gasPrice           = big.NewInt(10000000)
	callerAddress      = identityset.Address(30).String()
)

var (
	erc1155ABI         abi.ABI
	erc721ABI          abi.ABI
	Transfer           hash.Hash256
	HashTransferBatch  hash.Hash256
	HashTransferSingle hash.Hash256
)

func initAddress() error {
	var err error
	//Transfer(address,address,uint256)
	Transfer, err = hash.HexStringToHash256(TransferString)
	if err != nil {
		return err
	}
	HashTransferBatch, err = hash.HexStringToHash256(TransferBatchString)
	if err != nil {
		return err
	}
	HashTransferSingle, err = hash.HexStringToHash256(TransferSingleString)
	if err != nil {
		return err
	}

	erc1155ABI, err = abi.JSON(strings.NewReader(ERC1155ABI))
	if err != nil {
		return err
	}
	erc721ABI, err = abi.JSON(strings.NewReader(ERC721ABI))
	if err != nil {
		return err
	}
	return nil
}

func isSBT(addr string, ab abi.ABI) (bool, error) {
	cli := kernel.ChainClient()
	callData, _ := ab.Pack("supportsInterface", interfaceId)
	execution := action.NewExecution(addr, transferAmount, callData)
	request := &iotexapi.ReadContractRequest{
		Execution:     execution.Proto(),
		CallerAddress: callerAddress,
	}
	res, err := cli.ReadContract(context.Background(), request)
	if err != nil {
		return false, err
	}
	if res.Receipt.Status == successStatus {
		if len(res.Data) == 0 {
			return false, nil
		}
		data, err := hex.DecodeString(res.Data)
		if err != nil {
			return false, err
		}
		unPack, err := ab.Unpack("supportsInterface", data)
		if err != nil {
			return false, err
		}
		out0 := *abi.ConvertType(unPack[0], new(bool)).(*bool)
		return out0, nil
	}
	return false, errors.New("read erc1155 contract failed")
}
