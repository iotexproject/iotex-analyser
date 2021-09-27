package main

import (
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-core/action"
)

func getAddresFromHash256(h hash.Hash256) (address.Address, error) {
	hexStr := hex.EncodeToString(h[:])
	ethAddr := hexStr[24:]
	ethAddress := common.HexToAddress(ethAddr)
	return address.FromBytes(ethAddress.Bytes())
}

func getAddressFromHash256ByIndex(h []hash.Hash256, i int) (address.Address, error) {
	if i >= len(h) {
		return nil, errors.New("invalid index")
	}
	return getAddresFromHash256(h[i])
}

func getbigIntFromHash256ByIndex(h []hash.Hash256, i int) (*big.Int, error) {
	if i >= len(h) {
		return nil, errors.New("invalid index")
	}
	return new(big.Int).SetBytes(h[i][:]), nil
}

func UnpackLog(a abi.ABI, out interface{}, event string, log *action.Log) error {
	if len(log.Data) > 0 {
		if err := a.UnpackIntoInterface(out, event, log.Data); err != nil {
			return err
		}
	}
	var indexed abi.Arguments
	for _, arg := range a.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	topics := []common.Hash{}
	for _, topic := range log.Topics {
		topics = append(topics, common.BytesToHash(topic[:]))
	}
	return abi.ParseTopics(out, indexed, topics[1:])
}
