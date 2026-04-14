package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/iotexproject/go-pkgs/hash"
	dbpkg "github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var testGormDB *gorm.DB

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().Port(9989),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("WARNING: embedded postgres failed to start: %v\n  DB tests will be skipped.\n", err)
		os.Exit(m.Run())
	}

	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=9989 sslmode=disable"
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
	// candidate table is owned by another package; create it directly (singular).
	if testGormDB != nil {
		if err := testGormDB.Exec(`CREATE TABLE IF NOT EXISTS candidate (
			id               BIGSERIAL PRIMARY KEY,
			block_height     BIGINT       NOT NULL DEFAULT 0,
			operator_address VARCHAR(42)  NOT NULL DEFAULT '',
			name             VARCHAR(64)  NOT NULL DEFAULT ''
		)`).Error; err != nil {
			fmt.Printf("WARNING: failed to create candidate table: %v\n", err)
			testGormDB = nil
		}
	}
	// block_meta table
	if testGormDB != nil {
		if err := testGormDB.AutoMigrate(&models.BlockMeta{}); err != nil {
			fmt.Printf("WARNING: failed to migrate block_meta: %v\n", err)
			testGormDB = nil
		}
	}

	code := m.Run()
	pg.Stop()
	os.Exit(code)
}

// resetDB truncates all test data between tests.
func resetDB(t *testing.T) {
	t.Helper()
	if testGormDB == nil {
		t.Skip("embedded postgres not available")
	}
	testGormDB.Exec("TRUNCATE candidate RESTART IDENTITY CASCADE")
	testGormDB.Exec("TRUNCATE block_meta RESTART IDENTITY CASCADE")
	testGormDB.Exec("DELETE FROM index_heights WHERE name = 'block_meta'")
	dbpkg.ClearIndexCache()
}

// insertCandidate inserts a candidate row for testing.
func insertCandidate(t *testing.T, height uint64, operatorAddr, name string) {
	t.Helper()
	require.NoError(t, testGormDB.Exec(
		`INSERT INTO candidate (block_height, operator_address, name) VALUES (?, ?, ?)`,
		height, operatorAddr, name,
	).Error)
}

// makeTestBlock creates a minimal valid block at the given height.
// The producer address is always identityset.Address(0).
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

// newPlugin creates a fresh plugin with an empty candidateCache (no chain client needed).
func newPlugin(t *testing.T) *blockMetaPlugin {
	t.Helper()
	if testGormDB == nil {
		t.Skip("embedded postgres not available")
	}
	p := &blockMetaPlugin{
		candidateCache: map[string]string{},
	}
	// AutoMigrate is idempotent; height==0 drops+recreates the table on first call.
	if err := dbpkg.AutoMigrate(p.Name(), &models.BlockMeta{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return p
}

// queryBlockMetas returns all block_meta rows as map[height]BlockMeta.
func queryBlockMetas(t *testing.T) map[uint64]models.BlockMeta {
	t.Helper()
	var rows []models.BlockMeta
	require.NoError(t, testGormDB.Find(&rows).Error)
	m := make(map[uint64]models.BlockMeta, len(rows))
	for _, r := range rows {
		m[r.BlockHeight] = r
	}
	return m
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestLookupCandidateName_UsesCache verifies that lookupCandidateName returns
// the cached name on a hit and falls back to DB on a miss.
func TestLookupCandidateName_UsesCache(t *testing.T) {
	resetDB(t)
	require := require.New(t)

	producerAddr := identityset.Address(0).String()
	other := identityset.Address(1).String()
	insertCandidate(t, 1, producerAddr, "db-name")
	insertCandidate(t, 1, other, "other-name")

	p := newPlugin(t)
	// Cache hit: should return cached value, not the DB value.
	p.candidateCache[producerAddr] = "cached-name"
	require.Equal("cached-name", p.lookupCandidateName(100, producerAddr))

	// Cache miss: should fall back to DB.
	require.Equal("other-name", p.lookupCandidateName(100, other))
}

// TestGetCandidateNamesBatch verifies the batch SQL fetches the latest-by-id
// name per address as of the given height, excluding rows above the height.
func TestGetCandidateNamesBatch(t *testing.T) {
	resetDB(t)
	require := require.New(t)

	addrA := identityset.Address(1).String()
	addrB := identityset.Address(2).String()
	addrC := identityset.Address(3).String()

	// addrA: two rows at height 1 and 5 → at height 4, only "alice-old" visible.
	insertCandidate(t, 1, addrA, "alice-old")
	insertCandidate(t, 5, addrA, "alice-new")
	// addrB: one row at height 2.
	insertCandidate(t, 2, addrB, "bob")
	// addrC: row at height 6 → NOT visible at height 4.
	insertCandidate(t, 6, addrC, "carol")

	res, err := getCandidateNamesBatch(4, []string{addrA, addrB, addrC})
	require.NoError(err)
	require.Equal("alice-old", res[addrA])
	require.Equal("bob", res[addrB])
	require.NotContains(res, addrC, "carol's row is above height 4 and must be excluded")

	// At height 5, addrA shows updated name; at height 6, carol appears.
	res2, err := getCandidateNamesBatch(6, []string{addrA, addrC})
	require.NoError(err)
	require.Equal("alice-new", res2[addrA])
	require.Equal("carol", res2[addrC])
}

// TestCommit_WritesCorrectRows verifies that metas loaded into b.metas are
// written to the block_meta table by commit(), with the correct field values.
// We bypass putBlock (which needs the kernel rolldpos protocol) and populate
// b.metas directly to keep the test self-contained.
func TestCommit_WritesCorrectRows(t *testing.T) {
	resetDB(t)
	require := require.New(t)
	ctx := context.Background()

	producerAddr := identityset.Address(0).String()
	insertCandidate(t, 1, producerAddr, "validator-one")

	p := newPlugin(t)
	p.candidateCache = map[string]string{producerAddr: "validator-one"}

	// Simulate what putBlock would compute for two blocks.
	for _, h := range []uint64{2, 3} {
		_ = makeTestBlock(h) // ensure block can be constructed
		p.metas = append(p.metas, models.BlockMeta{
			BlockHeight:     h,
			ProducerAddress: producerAddr,
			ProducerName:    p.lookupCandidateName(h, producerAddr),
			BlockReward:     decimal.NewFromInt(int64(h) * 100),
			EpochReward:     decimal.Zero,
			FoundationBonus: decimal.Zero,
		})
	}
	p.tipHeight = 3
	require.NoError(p.commit())

	rows := queryBlockMetas(t)
	require.Len(rows, 2)
	for _, h := range []uint64{2, 3} {
		row, ok := rows[h]
		require.True(ok, "missing block_meta for height %d", h)
		require.Equal(producerAddr, row.ProducerAddress)
		require.Equal("validator-one", row.ProducerName)
		require.Equal(decimal.NewFromInt(int64(h)*100).String(), row.BlockReward.String())
	}

	// index height must be updated to tipHeight.
	gotHeight, err := dbpkg.GetIndexHeight("block_meta")
	require.NoError(err)
	require.Equal(uint64(3), gotHeight)
	_ = ctx
}

// TestCommit_BatchUpsert verifies that writing the same block height twice
// does not trigger a unique-constraint error (OnConflict{UpdateAll:true}).
func TestCommit_BatchUpsert(t *testing.T) {
	resetDB(t)
	require := require.New(t)

	producerAddr := identityset.Address(0).String()
	p := newPlugin(t)

	// First write.
	p.metas = []models.BlockMeta{{
		BlockHeight:     10,
		ProducerAddress: producerAddr,
		ProducerName:    "validator-one",
		EpochReward:     decimal.Zero,
		BlockReward:     decimal.Zero,
		FoundationBonus: decimal.Zero,
	}}
	p.tipHeight = 10
	require.NoError(p.commit())

	// Second write for the same height — should upsert.
	p2 := newPlugin(t)
	p2.metas = []models.BlockMeta{{
		BlockHeight:     10,
		ProducerAddress: producerAddr,
		ProducerName:    "validator-one-updated",
		EpochReward:     decimal.Zero,
		BlockReward:     decimal.Zero,
		FoundationBonus: decimal.Zero,
	}}
	p2.tipHeight = 10
	require.NoError(p2.commit())

	rows := queryBlockMetas(t)
	require.Len(rows, 1)
	require.Equal("validator-one-updated", rows[10].ProducerName)
}
