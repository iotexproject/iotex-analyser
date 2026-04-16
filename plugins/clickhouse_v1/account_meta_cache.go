package main

import (
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/action"
)

const accountMetaLookupBatchSize = 1000

type accountContractLookup func([]string) (map[string]bool, error)

func collectUnknownTransactionSenders(receipts []*action.Receipt, cache map[string]bool) []string {
	seen := make(map[string]struct{})
	addrs := make([]string, 0)
	for _, receipt := range receipts {
		for _, txLog := range receipt.TransactionLogs() {
			if txLog.Sender == "" {
				continue
			}
			if _, ok := cache[txLog.Sender]; ok {
				continue
			}
			if _, ok := seen[txLog.Sender]; ok {
				continue
			}
			seen[txLog.Sender] = struct{}{}
			addrs = append(addrs, txLog.Sender)
		}
	}
	return addrs
}

func warmAccountContractCache(receipts []*action.Receipt, cache map[string]bool, lookup accountContractLookup) error {
	unknown := collectUnknownTransactionSenders(receipts, cache)
	for start := 0; start < len(unknown); start += accountMetaLookupBatchSize {
		end := start + accountMetaLookupBatchSize
		if end > len(unknown) {
			end = len(unknown)
		}
		batch := unknown[start:end]
		flags, err := lookup(batch)
		if err != nil {
			return err
		}
		for _, addr := range batch {
			cache[addr] = flags[addr]
		}
	}
	return nil
}

func loadAccountContractFlags(addrs []string) (map[string]bool, error) {
	return models.LoadAccountContractFlags(addrs)
}
