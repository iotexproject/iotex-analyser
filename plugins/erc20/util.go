package main

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/iotex-core/action"
)

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
