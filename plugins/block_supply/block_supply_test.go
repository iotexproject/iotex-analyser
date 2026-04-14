package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	dbpkg "github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var testGormDB *gorm.DB

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().Port(9988),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("WARNING: embedded postgres failed to start: %v\n  DB tests will be skipped.\n", err)
		os.Exit(m.Run())
	}

	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=9988 sslmode=disable"
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

	if err := testGormDB.AutoMigrate(&dbpkg.IndexHeight{}, &dbpkg.Store{}); err != nil {
		fmt.Printf("WARNING: failed to migrate infra tables: %v\n", err)
		testGormDB = nil
	}
	// account_income is owned by another plugin package; create the table directly.
	if testGormDB != nil {
		if err := testGormDB.Exec(`CREATE TABLE IF NOT EXISTS account_income (
			id            BIGSERIAL PRIMARY KEY,
			block_height  BIGINT        NOT NULL DEFAULT 0,
			address       VARCHAR(42)   NOT NULL DEFAULT '',
			in_flow       DECIMAL(42,0) NOT NULL DEFAULT 0,
			out_flow      DECIMAL(42,0) NOT NULL DEFAULT 0,
			in_num_actions  INT         NOT NULL DEFAULT 0,
			out_num_actions INT         NOT NULL DEFAULT 0
		)`).Error; err != nil {
			fmt.Printf("WARNING: failed to create account_income table: %v\n", err)
			testGormDB = nil
		}
	}

	code := m.Run()
	pg.Stop()
	os.Exit(code)
}

// resetDB clears all test data between tests.
func resetDB(t *testing.T) {
	t.Helper()
	if testGormDB == nil {
		t.Skip("embedded postgres not available")
	}
	testGormDB.Exec("TRUNCATE account_income RESTART IDENTITY CASCADE")
	testGormDB.Exec("DROP TABLE IF EXISTS block_supply")
	testGormDB.Exec("DELETE FROM index_heights WHERE name = 'block_supply'")
	dbpkg.ClearIndexCache()
}

// insertAIRow inserts a synthetic account_income row for block_supply to query.
func insertAIRow(t *testing.T, height uint64, addr string, inFlow, outFlow *big.Int) {
	t.Helper()
	require.NoError(t, testGormDB.Exec(
		`INSERT INTO account_income (block_height, address, in_flow, out_flow, in_num_actions, out_num_actions)
		 VALUES (?, ?, ?, ?, 1, 1)`,
		height, addr, inFlow.String(), outFlow.String(),
	).Error)
}

// makeTestBlock creates a minimal valid block at the given height.
func makeTestBlock(height uint64) *block.Block {
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
	return &blk
}

// newPlugin creates and starts a fresh plugin backed by the test DB.
func newPlugin(t *testing.T) *blockSupplyPlugin {
	t.Helper()
	p := &blockSupplyPlugin{}
	require.NoError(t, p.Start(context.Background()))
	return p
}

// supplyFromFullAggregate computes TotalSupply/CirculatingSupply at height
// using the pre-optimization full-aggregate approach (accountBalanceByHeight
// with block_height<=N), which is the reference/oracle for consistency checks.
func supplyFromFullAggregate(t *testing.T, height uint64) (totalSupply, circulating string) {
	t.Helper()
	zeroAddrBal, err := accountBalanceByHeight(height, address.ZeroAddress)
	require.NoError(t, err)
	ts := computeTotalSupply(zeroAddrBal)
	lockedBal, err := accountBalanceByHeight(height, lockAddresses)
	require.NoError(t, err)
	cs, err := computeTotalCirculatingSupply(ts, lockedBal)
	require.NoError(t, err)
	return ts, cs
}

// queryBlockSupply fetches all block_supply rows as a map[height]BlockSupply.
func queryBlockSupply(t *testing.T) map[uint64]models.BlockSupply {
	t.Helper()
	var rows []models.BlockSupply
	require.NoError(t, testGormDB.Find(&rows).Error)
	m := make(map[uint64]models.BlockSupply, len(rows))
	for _, r := range rows {
		m[r.BlockHeight] = r
	}
	return m
}

// TestRunningBalance_InvariantHolds verifies the core mathematical invariant:
// after processing each block, runningBalance[addr] == accountBalanceByHeight(height, addr).
// This guarantees the new implementation produces identical values to the old full-aggregate approach.
func TestRunningBalance_InvariantHolds(t *testing.T) {
	resetDB(t)
	require := require.New(t)
	ctx := context.Background()

	// Test data: 4 blocks with activity on tracked addresses.
	//   height 1: zero addr receives 1000 (burned)
	//   height 2: lock addr receives 500 (locked)
	//   height 3: zero addr sends 200 (un-burned), lock addr receives 100 more
	//   height 4: zero addr receives 300 (burned)
	insertAIRow(t, 1, address.ZeroAddress, big.NewInt(1000), big.NewInt(0))
	insertAIRow(t, 2, lockAddresses, big.NewInt(500), big.NewInt(0))
	insertAIRow(t, 3, address.ZeroAddress, big.NewInt(0), big.NewInt(200))
	insertAIRow(t, 3, lockAddresses, big.NewInt(100), big.NewInt(0))
	insertAIRow(t, 4, address.ZeroAddress, big.NewInt(300), big.NewInt(0))

	p := newPlugin(t)
	// Verify invariant: runningBalance drift remains zero after each block.
	for _, h := range []uint64{1, 2, 3, 4} {
		require.NoError(p.putBlock(ctx, makeTestBlock(h)))

		for _, addr := range []string{address.ZeroAddress, lockAddresses} {
			cumulative, err := accountBalanceByHeight(h, addr)
			require.NoError(err)
			require.Equal(
				cumulative.String(),
				p.runningBalance[addr].String(),
				"runningBalance mismatch for addr %s at height %d", addr, h,
			)
		}
	}
}

// TestComputeSupply_MatchesFullAggregate verifies that the values computed via
// running balances (new approach) equal those from the full aggregate query (old approach).
func TestComputeSupply_MatchesFullAggregate(t *testing.T) {
	resetDB(t)
	require := require.New(t)
	ctx := context.Background()

	insertAIRow(t, 1, address.ZeroAddress, big.NewInt(1000), big.NewInt(0))
	insertAIRow(t, 2, lockAddresses, big.NewInt(500), big.NewInt(0))
	insertAIRow(t, 3, address.ZeroAddress, big.NewInt(0), big.NewInt(200))
	insertAIRow(t, 3, lockAddresses, big.NewInt(100), big.NewInt(0))
	insertAIRow(t, 4, address.ZeroAddress, big.NewInt(300), big.NewInt(0))

	p := newPlugin(t)
	for _, h := range []uint64{1, 2, 3, 4} {
		require.NoError(p.putBlock(ctx, makeTestBlock(h)))
	}

	// p.supplies now holds the computed results. Compare each against oracle.
	require.Len(p.supplies, 4)
	for _, s := range p.supplies {
		wantTS, wantCS := supplyFromFullAggregate(t, s.BlockHeight)
		require.Equal(wantTS, s.TotalSupply,
			"TotalSupply mismatch at height %d", s.BlockHeight)
		require.Equal(wantCS, s.TotalCirculatingSupply,
			"TotalCirculatingSupply mismatch at height %d", s.BlockHeight)
	}
}

// TestPutBlocks_DBWritesMatchOracle verifies end-to-end: after PutBlocks, the rows
// written to the block_supply table match the full-aggregate oracle for every height.
func TestPutBlocks_DBWritesMatchOracle(t *testing.T) {
	resetDB(t)
	require := require.New(t)
	ctx := context.Background()

	insertAIRow(t, 1, address.ZeroAddress, big.NewInt(1000), big.NewInt(0))
	insertAIRow(t, 2, lockAddresses, big.NewInt(500), big.NewInt(0))
	insertAIRow(t, 3, address.ZeroAddress, big.NewInt(0), big.NewInt(200))
	insertAIRow(t, 3, lockAddresses, big.NewInt(100), big.NewInt(0))
	insertAIRow(t, 4, address.ZeroAddress, big.NewInt(300), big.NewInt(0))

	blks := []*block.Block{
		makeTestBlock(1), makeTestBlock(2), makeTestBlock(3), makeTestBlock(4),
	}

	p := newPlugin(t)
	require.NoError(p.PutBlocks(ctx, blks))

	rows := queryBlockSupply(t)
	require.Len(rows, 4)

	for _, h := range []uint64{1, 2, 3, 4} {
		row, ok := rows[h]
		require.True(ok, "missing block_supply row for height %d", h)
		wantTS, wantCS := supplyFromFullAggregate(t, h)
		require.Equal(wantTS, row.TotalSupply,
			"TotalSupply mismatch at height %d", h)
		require.Equal(wantCS, row.TotalCirculatingSupply,
			"TotalCirculatingSupply mismatch at height %d", h)
	}
}

// TestPutBlocks_vs_PutBlock_DBConsistency verifies that batch (PutBlocks) and
// sequential (PutBlock one by one) produce identical block_supply DB rows.
func TestPutBlocks_vs_PutBlock_DBConsistency(t *testing.T) {
	insertRows := func() {
		insertAIRow(t, 1, address.ZeroAddress, big.NewInt(500), big.NewInt(0))
		insertAIRow(t, 2, lockAddresses, big.NewInt(300), big.NewInt(0))
		insertAIRow(t, 2, address.ZeroAddress, big.NewInt(0), big.NewInt(100))
		insertAIRow(t, 3, lockAddresses, big.NewInt(0), big.NewInt(50))
		insertAIRow(t, 3, address.ZeroAddress, big.NewInt(200), big.NewInt(0))
	}
	blks := []*block.Block{makeTestBlock(1), makeTestBlock(2), makeTestBlock(3)}
	ctx := context.Background()
	require := require.New(t)

	// --- Batch: PutBlocks ---
	resetDB(t)
	insertRows()
	pBatch := newPlugin(t)
	require.NoError(pBatch.PutBlocks(ctx, blks))
	batchRows := queryBlockSupply(t)

	// --- Sequential: PutBlock one per block, new plugin instance each time ---
	resetDB(t)
	insertRows()
	for _, blk := range blks {
		p := newPlugin(t)
		require.NoError(p.PutBlock(ctx, blk))
	}
	seqRows := queryBlockSupply(t)

	require.Equal(len(batchRows), len(seqRows))
	for h, batchRow := range batchRows {
		seqRow, ok := seqRows[h]
		require.True(ok, "height %d missing from sequential result", h)
		require.Equal(batchRow.TotalSupply, seqRow.TotalSupply,
			"TotalSupply mismatch at height %d", h)
		require.Equal(batchRow.TotalCirculatingSupply, seqRow.TotalCirculatingSupply,
			"TotalCirculatingSupply mismatch at height %d", h)
	}
}
