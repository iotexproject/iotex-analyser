package main

import (
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/iotex-address/address"
)

const (
	// transferSha3 is sha3 of xrc20's transfer event,keccak('Transfer(address,address,uint256)')
	transferSha3 = "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	topicsPlusDataLen = 256
	sha3Len           = 64
	contractParamsLen = 64
	addressLen        = 40
	successStatus     = uint64(1)
	revertStatus      = uint64(106)
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
