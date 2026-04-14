package main

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
	"github.com/stretchr/testify/require"
)

func makeBlock(height uint64, txLogs [][]*action.TransactionLog) *block.Block {
	tb := block.TestingBuilder{}
	blk, err := tb.
		SetPrevBlockHash(hash.ZeroHash256).
		SetVersion(1).
		SetTimeStamp(time.Now()).
		SetHeight(height).
		SignAndBuild(identityset.PrivateKey(0))
	if err != nil {
		panic(err)
	}
	for _, logs := range txLogs {
		r := &action.Receipt{}
		r = r.AddTransactionLogs(logs...)
		blk.Receipts = append(blk.Receipts, r)
	}
	return &blk
}

var (
	addrA = identityset.Address(1).String()
	addrB = identityset.Address(2).String()
	addrC = identityset.Address(3).String()
)

func TestGetIncomes_Basic(t *testing.T) {
	require := require.New(t)
	blk := makeBlock(1, [][]*action.TransactionLog{
		{{Sender: addrA, Recipient: addrB, Amount: big.NewInt(100)}},
		{{Sender: addrB, Recipient: addrC, Amount: big.NewInt(50)}},
	})
	incomes, err := getIncomes(blk)
	require.NoError(err)
	require.Equal("0", incomes[addrA].inFlow.String())
	require.Equal("100", incomes[addrA].outFlow.String())
	require.Equal(1, incomes[addrA].outNumActions)
	require.Equal(0, incomes[addrA].inNumActions)
	require.Equal("100", incomes[addrB].inFlow.String())
	require.Equal("50", incomes[addrB].outFlow.String())
	require.Equal(1, incomes[addrB].inNumActions)
	require.Equal(1, incomes[addrB].outNumActions)
	require.Equal("50", incomes[addrC].inFlow.String())
	require.Equal("0", incomes[addrC].outFlow.String())
	require.Equal(1, incomes[addrC].inNumActions)
	require.Equal(0, incomes[addrC].outNumActions)
}

func TestGetIncomes_MultiTransferSameAddress(t *testing.T) {
	require := require.New(t)
	blk := makeBlock(1, [][]*action.TransactionLog{
		{
			{Sender: addrA, Recipient: addrB, Amount: big.NewInt(100)},
			{Sender: addrA, Recipient: addrC, Amount: big.NewInt(200)},
			{Sender: addrB, Recipient: addrA, Amount: big.NewInt(30)},
		},
	})
	incomes, err := getIncomes(blk)
	require.NoError(err)
	require.Equal("30", incomes[addrA].inFlow.String())
	require.Equal("300", incomes[addrA].outFlow.String())
	require.Equal(2, incomes[addrA].outNumActions)
	require.Equal(1, incomes[addrA].inNumActions)
	require.Equal("100", incomes[addrB].inFlow.String())
	require.Equal("30", incomes[addrB].outFlow.String())
}

func TestGetIncomes_EmptyBlock(t *testing.T) {
	require := require.New(t)
	blk := makeBlock(1, nil)
	incomes, err := getIncomes(blk)
	require.NoError(err)
	require.Empty(incomes)
}

func TestPutBlock_InMemoryAccumulation(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	p := &accountIncomePlugin{
		accountIncomeCountMap: make(map[string]*AccountIncomeCount),
	}
	blk1 := makeBlock(1, [][]*action.TransactionLog{
		{{Sender: addrA, Recipient: addrB, Amount: big.NewInt(100)}},
	})
	blk2 := makeBlock(2, [][]*action.TransactionLog{
		{{Sender: addrB, Recipient: addrC, Amount: big.NewInt(40)}},
		{{Sender: addrA, Recipient: addrC, Amount: big.NewInt(60)}},
	})
	require.NoError(p.putBlock(ctx, blk1))
	require.NoError(p.putBlock(ctx, blk2))
	aCount := p.accountIncomeCountMap[addrA]
	require.NotNil(aCount)
	require.Equal("160", aCount.OutFlow.String())
	require.Equal("0", aCount.InFlow.String())
	bCount := p.accountIncomeCountMap[addrB]
	require.NotNil(bCount)
	require.Equal("100", bCount.InFlow.String())
	require.Equal("40", bCount.OutFlow.String())
	cCount := p.accountIncomeCountMap[addrC]
	require.NotNil(cCount)
	require.Equal("100", cCount.InFlow.String())
	require.Equal("0", cCount.OutFlow.String())
	require.Len(p.accountIncome, 5)
}

func TestPutBlocksConsistency(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	blk1 := makeBlock(1, [][]*action.TransactionLog{
		{
			{Sender: addrA, Recipient: addrB, Amount: big.NewInt(500)},
			{Sender: addrA, Recipient: addrC, Amount: big.NewInt(200)},
		},
	})
	blk2 := makeBlock(2, [][]*action.TransactionLog{
		{{Sender: addrB, Recipient: addrA, Amount: big.NewInt(100)}},
	})
	blk3 := makeBlock(3, [][]*action.TransactionLog{
		{{Sender: addrC, Recipient: addrA, Amount: big.NewInt(50)}},
		{{Sender: addrA, Recipient: addrB, Amount: big.NewInt(300)}},
	})
	pSeq := &accountIncomePlugin{accountIncomeCountMap: make(map[string]*AccountIncomeCount)}
	for _, blk := range []*block.Block{blk1, blk2, blk3} {
		require.NoError(pSeq.putBlock(ctx, blk))
	}
	pBatch := &accountIncomePlugin{accountIncomeCountMap: make(map[string]*AccountIncomeCount)}
	for _, blk := range []*block.Block{blk1, blk2, blk3} {
		require.NoError(pBatch.putBlock(ctx, blk))
	}
	require.Equal(len(pSeq.accountIncomeCountMap), len(pBatch.accountIncomeCountMap))
	for addr, seqCount := range pSeq.accountIncomeCountMap {
		batchCount := pBatch.accountIncomeCountMap[addr]
		require.NotNil(batchCount, "address %s missing in batch result", addr)
		require.Equal(seqCount.InFlow.String(), batchCount.InFlow.String(), "InFlow mismatch for %s", addr)
		require.Equal(seqCount.OutFlow.String(), batchCount.OutFlow.String(), "OutFlow mismatch for %s", addr)
		require.Equal(seqCount.InNumActions, batchCount.InNumActions, "InNumActions mismatch for %s", addr)
		require.Equal(seqCount.OutNumActions, batchCount.OutNumActions, "OutNumActions mismatch for %s", addr)
	}
	aSeq := pSeq.accountIncomeCountMap[addrA]
	require.Equal("150", aSeq.InFlow.String())
	require.Equal("1000", aSeq.OutFlow.String())
	bSeq := pSeq.accountIncomeCountMap[addrB]
	require.Equal("800", bSeq.InFlow.String())
	require.Equal("100", bSeq.OutFlow.String())
	cSeq := pSeq.accountIncomeCountMap[addrC]
	require.Equal("200", cSeq.InFlow.String())
	require.Equal("50", cSeq.OutFlow.String())
}
