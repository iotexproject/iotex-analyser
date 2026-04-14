package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	dbpkg "github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var testGormDB *gorm.DB

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().Port(9987),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("WARNING: embedded postgres failed to start: %v\n  DB tests will be skipped.\n", err)
		os.Exit(m.Run())
	}

	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=9987 sslmode=disable"
	var err error
	testGormDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		fmt.Printf("WARNING: failed to connect to embedded postgres: %v\n  DB tests will be skipped.\n", err)
		code := m.Run()
		pg.Stop()
		os.Exit(code)
	}

	dbpkg.SetDB(testGormDB)

	if err := testGormDB.AutoMigrate(
		&dbpkg.IndexHeight{}, &dbpkg.Store{},
		&AccountIncome{}, &AccountIncomeCount{},
	); err != nil {
		fmt.Printf("WARNING: failed to migrate tables: %v\n  DB tests will be skipped.\n", err)
		testGormDB = nil
	}

	code := m.Run()
	pg.Stop()
	os.Exit(code)
}

func resetTestDB(t *testing.T) {
	t.Helper()
	if testGormDB == nil {
		t.Skip("embedded postgres not available")
	}
	testGormDB.Exec("TRUNCATE account_income RESTART IDENTITY CASCADE")
	testGormDB.Exec("TRUNCATE account_income_count RESTART IDENTITY CASCADE")
	testGormDB.Exec("DELETE FROM index_heights WHERE name = 'account_income'")
	dbpkg.ClearIndexCache()
}

func queryIncomeCountMap(t *testing.T) map[string]*AccountIncomeCount {
	t.Helper()
	var rows []AccountIncomeCount
	require.NoError(t, testGormDB.Find(&rows).Error)
	result := make(map[string]*AccountIncomeCount, len(rows))
	for i := range rows {
		r := rows[i]
		result[r.Address] = &r
	}
	return result
}

// TestPutBlock_DBCommit verifies that commit() writes the correct rows to
// account_income and account_income_count.
func TestPutBlock_DBCommit(t *testing.T) {
	resetTestDB(t)
	require := require.New(t)
	ctx := context.Background()

	p := &accountIncomePlugin{
		accountIncomeCountMap: make(map[string]*AccountIncomeCount),
	}
	// blk1: addrA -> addrB  300
	blk1 := makeBlock(1, [][]*action.TransactionLog{
		{{Sender: addrA, Recipient: addrB, Amount: big.NewInt(300)}},
	})
	// blk2: addrB -> addrC  100,  addrA -> addrC  200
	blk2 := makeBlock(2, [][]*action.TransactionLog{
		{{Sender: addrB, Recipient: addrC, Amount: big.NewInt(100)}},
		{{Sender: addrA, Recipient: addrC, Amount: big.NewInt(200)}},
	})

	require.NoError(p.putBlock(ctx, blk1))
	require.NoError(p.putBlock(ctx, blk2))
	p.tipHeight = 2
	require.NoError(p.commit())

	// account_income: 2 rows from blk1 + 3 rows from blk2 = 5
	var aiCount int64
	require.NoError(testGormDB.Model(&AccountIncome{}).Count(&aiCount).Error)
	require.Equal(int64(5), aiCount)

	counts := queryIncomeCountMap(t)
	require.Len(counts, 3)

	require.Equal("0", counts[addrA].InFlow.String())
	require.Equal("500", counts[addrA].OutFlow.String())
	require.Equal(0, counts[addrA].InNumActions)
	require.Equal(2, counts[addrA].OutNumActions)

	require.Equal("300", counts[addrB].InFlow.String())
	require.Equal("100", counts[addrB].OutFlow.String())
	require.Equal(1, counts[addrB].InNumActions)
	require.Equal(1, counts[addrB].OutNumActions)

	require.Equal("300", counts[addrC].InFlow.String())
	require.Equal("0", counts[addrC].OutFlow.String())
	require.Equal(2, counts[addrC].InNumActions)
	require.Equal(0, counts[addrC].OutNumActions)
}

// TestPutBlock_OnConflictAccumulates verifies that successive PutBlock calls
// (each with its own commit) correctly accumulate values in account_income_count
// via ON CONFLICT DO UPDATE, rather than replacing existing rows.
func TestPutBlock_OnConflictAccumulates(t *testing.T) {
	resetTestDB(t)
	require := require.New(t)
	ctx := context.Background()

	// blk1: addrA -> addrB  500
	blk1 := makeBlock(1, [][]*action.TransactionLog{
		{{Sender: addrA, Recipient: addrB, Amount: big.NewInt(500)}},
	})
	// blk2: addrB -> addrA  200,  addrC -> addrA  50  (one receipt, two logs)
	blk2 := makeBlock(2, [][]*action.TransactionLog{
		{
			{Sender: addrB, Recipient: addrA, Amount: big.NewInt(200)},
			{Sender: addrC, Recipient: addrA, Amount: big.NewInt(50)},
		},
	})
	// blk3: addrA -> addrC  100
	blk3 := makeBlock(3, [][]*action.TransactionLog{
		{{Sender: addrA, Recipient: addrC, Amount: big.NewInt(100)}},
	})

	// Each PutBlock creates a fresh plugin (no shared in-memory state),
	// forcing the ON CONFLICT path for subsequent blocks.
	for _, blk := range []*block.Block{blk1, blk2, blk3} {
		p := &accountIncomePlugin{accountIncomeCountMap: make(map[string]*AccountIncomeCount)}
		require.NoError(p.PutBlock(ctx, blk))
	}

	counts := queryIncomeCountMap(t)
	require.Len(counts, 3)

	// addrA: received 200+50=250, sent 500+100=600
	require.Equal("250", counts[addrA].InFlow.String())
	require.Equal("600", counts[addrA].OutFlow.String())
	require.Equal(2, counts[addrA].InNumActions)
	require.Equal(2, counts[addrA].OutNumActions)

	// addrB: received 500, sent 200
	require.Equal("500", counts[addrB].InFlow.String())
	require.Equal("200", counts[addrB].OutFlow.String())
	require.Equal(1, counts[addrB].InNumActions)
	require.Equal(1, counts[addrB].OutNumActions)

	// addrC: received 100, sent 50
	require.Equal("100", counts[addrC].InFlow.String())
	require.Equal("50", counts[addrC].OutFlow.String())
	require.Equal(1, counts[addrC].InNumActions)
	require.Equal(1, counts[addrC].OutNumActions)
}

// TestPutBlocks_vs_PutBlock_DBConsistency verifies that PutBlocks (one batch commit)
// and sequential PutBlock calls (one commit per block) produce identical DB state.
func TestPutBlocks_vs_PutBlock_DBConsistency(t *testing.T) {
	resetTestDB(t)
	require := require.New(t)
	ctx := context.Background()

	blk1 := makeBlock(1, [][]*action.TransactionLog{
		{
			{Sender: addrA, Recipient: addrB, Amount: big.NewInt(1000)},
			{Sender: addrA, Recipient: addrC, Amount: big.NewInt(400)},
		},
	})
	blk2 := makeBlock(2, [][]*action.TransactionLog{
		{{Sender: addrB, Recipient: addrA, Amount: big.NewInt(300)}},
	})
	blk3 := makeBlock(3, [][]*action.TransactionLog{
		{
			{Sender: addrC, Recipient: addrB, Amount: big.NewInt(150)},
			{Sender: addrA, Recipient: addrC, Amount: big.NewInt(200)},
		},
	})
	blks := []*block.Block{blk1, blk2, blk3}

	// --- Batch: PutBlocks (one commit for all 3 blocks) ---
	p1 := &accountIncomePlugin{accountIncomeCountMap: make(map[string]*AccountIncomeCount)}
	require.NoError(p1.PutBlocks(ctx, blks))
	batchCounts := queryIncomeCountMap(t)
	var batchAICount int64
	require.NoError(testGormDB.Model(&AccountIncome{}).Count(&batchAICount).Error)

	// --- Sequential: PutBlock one at a time ---
	resetTestDB(t)
	for _, blk := range blks {
		p := &accountIncomePlugin{accountIncomeCountMap: make(map[string]*AccountIncomeCount)}
		require.NoError(p.PutBlock(ctx, blk))
	}
	seqCounts := queryIncomeCountMap(t)
	var seqAICount int64
	require.NoError(testGormDB.Model(&AccountIncome{}).Count(&seqAICount).Error)

	// Same number of account_income rows
	require.Equal(batchAICount, seqAICount)

	// Identical account_income_count for every address
	require.Equal(len(batchCounts), len(seqCounts))
	for addr, batchRow := range batchCounts {
		seqRow := seqCounts[addr]
		require.NotNil(seqRow, "address %s missing in sequential result", addr)
		require.Equal(batchRow.InFlow.String(), seqRow.InFlow.String(), "InFlow mismatch for %s", addr)
		require.Equal(batchRow.OutFlow.String(), seqRow.OutFlow.String(), "OutFlow mismatch for %s", addr)
		require.Equal(batchRow.InNumActions, seqRow.InNumActions, "InNumActions mismatch for %s", addr)
		require.Equal(batchRow.OutNumActions, seqRow.OutNumActions, "OutNumActions mismatch for %s", addr)
	}
}
