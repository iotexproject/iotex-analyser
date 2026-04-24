package main

import (
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/stretchr/testify/require"
)

func receiptWithTransactionLogs(logs ...*action.TransactionLog) *action.Receipt {
	r := &action.Receipt{}
	return r.AddTransactionLogs(logs...)
}

func TestWarmAccountContractCacheDeduplicatesAndCachesMisses(t *testing.T) {
	addrA := identityset.Address(1).String()
	addrB := identityset.Address(2).String()
	addrC := identityset.Address(3).String()

	receipts := []*action.Receipt{
		receiptWithTransactionLogs(
			&action.TransactionLog{Sender: addrA},
			&action.TransactionLog{Sender: addrB},
			&action.TransactionLog{Sender: addrA},
		),
		receiptWithTransactionLogs(
			&action.TransactionLog{Sender: addrC},
			&action.TransactionLog{Sender: ""},
		),
	}

	cache := map[string]bool{addrA: true}
	lookedUp := make([][]string, 0)
	err := warmAccountContractCache(receipts, cache, func(addrs []string) (map[string]bool, error) {
		cp := append([]string(nil), addrs...)
		lookedUp = append(lookedUp, cp)
		return map[string]bool{
			addrB: true,
		}, nil
	})
	require.NoError(t, err)
	require.Equal(t, [][]string{{addrB, addrC}}, lookedUp)
	require.True(t, cache[addrA])
	require.True(t, cache[addrB])
	require.False(t, cache[addrC])
}

func TestHandleTransactionLogsUsesCachedContractFlags(t *testing.T) {
	addrA := identityset.Address(1).String()
	addrB := identityset.Address(2).String()
	addrC := identityset.Address(3).String()

	rows, err := handleTransactionLogs([]*action.TransactionLog{
		{
			Type:      iotextypes.TransactionLogType_NATIVE_TRANSFER,
			Sender:    addrA,
			Recipient: addrB,
			Amount:    big.NewInt(100),
		},
		{
			Type:      iotextypes.TransactionLogType_IN_CONTRACT_TRANSFER,
			Sender:    addrC,
			Recipient: addrB,
			Amount:    big.NewInt(50),
		},
	}, "abc", 12, time.Now().Unix(), map[string]bool{
		addrA: true,
		addrC: false,
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.True(t, rows[0].Internal)
	require.False(t, rows[1].Internal)
	require.Equal(t, "transfer", rows[0].Type)
	require.Equal(t, "execution", rows[1].Type)
}

func TestWarmAccountContractCacheBatchesLargeInput(t *testing.T) {
	receipts := make([]*action.Receipt, 0, accountMetaLookupBatchSize+5)
	expected := make([]string, 0, accountMetaLookupBatchSize+5)
	for i := 0; i < accountMetaLookupBatchSize+5; i++ {
		addr := fmt.Sprintf("sender-%d", i)
		expected = append(expected, addr)
		receipts = append(receipts, receiptWithTransactionLogs(&action.TransactionLog{Sender: addr}))
	}

	cache := make(map[string]bool)
	batchSizes := make([]int, 0)
	err := warmAccountContractCache(receipts, cache, func(addrs []string) (map[string]bool, error) {
		batchSizes = append(batchSizes, len(addrs))
		flags := make(map[string]bool, len(addrs))
		for _, addr := range addrs {
			flags[addr] = true
		}
		return flags, nil
	})
	require.NoError(t, err)
	require.Equal(t, []int{accountMetaLookupBatchSize, 5}, batchSizes)
	for _, addr := range expected {
		require.True(t, cache[addr])
	}
}

func TestWarmAccountContractCachePropagatesLookupError(t *testing.T) {
	addrA := identityset.Address(1).String()
	errExpected := errors.New("lookup failed")
	err := warmAccountContractCache([]*action.Receipt{
		receiptWithTransactionLogs(&action.TransactionLog{Sender: addrA}),
	}, map[string]bool{}, func(addrs []string) (map[string]bool, error) {
		return nil, errExpected
	})
	require.ErrorIs(t, err, errExpected)
}

func TestPutBlockReusesContractCacheAcrossBlocks(t *testing.T) {
	addrA := identityset.Address(1).String()
	addrB := identityset.Address(2).String()
	addrC := identityset.Address(3).String()

	lookupCalls := make([][]string, 0)
	p := &clickhouseV1Plugin{
		accountContract: make(map[string]bool),
		accountLookup: func(addrs []string) (map[string]bool, error) {
			cp := append([]string(nil), addrs...)
			lookupCalls = append(lookupCalls, cp)
			return map[string]bool{addrA: true, addrC: false}, nil
		},
	}

	require.NoError(t, p.putBlock(nil, makeTxLogBlock(1,
		[]*action.TransactionLog{{Sender: addrA, Recipient: addrB, Amount: big.NewInt(10)}},
	)))
	require.NoError(t, p.putBlock(nil, makeTxLogBlock(2,
		[]*action.TransactionLog{{Sender: addrA, Recipient: addrC, Amount: big.NewInt(5)}},
		[]*action.TransactionLog{{Sender: addrC, Recipient: addrB, Amount: big.NewInt(2)}},
	)))

	require.Equal(t, [][]string{{addrA}, {addrC}}, lookupCalls)
	require.Len(t, p.transactionLogs, 3)
	require.True(t, p.transactionLogs[0].Internal)
	require.True(t, p.transactionLogs[1].Internal)
	require.False(t, p.transactionLogs[2].Internal)
}

func TestPutBlockPropagatesAccountLookupError(t *testing.T) {
	addrA := identityset.Address(1).String()
	p := &clickhouseV1Plugin{
		accountContract: make(map[string]bool),
		accountLookup: func(addrs []string) (map[string]bool, error) {
			return nil, errors.New("boom")
		},
	}
	err := p.putBlock(nil, makeTxLogBlock(1,
		[]*action.TransactionLog{{Sender: addrA, Amount: big.NewInt(1)}},
	))
	require.ErrorContains(t, err, "failed to warm account contract cache")
}

func makeTxLogBlock(height uint64, txLogGroups ...[]*action.TransactionLog) *block.Block {
	tb := block.TestingBuilder{}
	blk, err := tb.
		SetPrevBlockHash(hash.ZeroHash256).
		SetVersion(1).
		SetTimeStamp(time.Unix(int64(height), 0)).
		SetHeight(height).
		SignAndBuild(identityset.PrivateKey(0))
	if err != nil {
		panic(err)
	}
	for i, logs := range txLogGroups {
		r := &action.Receipt{}
		var actionHash hash.Hash256
		actionHash[0] = byte(i + 1)
		actionHash[31] = byte(i + 1)
		r.ActionHash = actionHash
		r = r.AddTransactionLogs(logs...)
		blk.Receipts = append(blk.Receipts, r)
	}
	return &blk
}
