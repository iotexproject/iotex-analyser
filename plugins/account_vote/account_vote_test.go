package main

import (
	"fmt"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	dbpkg "github.com/iotexproject/iotex-analyser/db"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var testGormDB *gorm.DB

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().Port(9990),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("WARNING: embedded postgres failed to start: %v\n  DB tests will be skipped.\n", err)
		os.Exit(m.Run())
	}

	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=9990 sslmode=disable"
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

	if err := testGormDB.AutoMigrate(&dbpkg.IndexHeight{}, &dbpkg.Store{}, &AccountVote{}); err != nil {
		fmt.Printf("WARNING: failed to migrate tables: %v\n", err)
		testGormDB = nil
	}

	code := m.Run()
	pg.Stop()
	os.Exit(code)
}

// resetDB truncates account_vote and resets the identity sequence between tests.
func resetDB(t *testing.T) {
	t.Helper()
	if testGormDB == nil {
		t.Skip("embedded postgres not available")
	}
	testGormDB.Exec("TRUNCATE account_vote RESTART IDENTITY CASCADE")
}

// insertVote is a helper that inserts an AccountVote row and returns the auto-assigned ID.
func insertVote(t *testing.T, v AccountVote) uint64 {
	t.Helper()
	require.NoError(t, testGormDB.Create(&v).Error)
	return v.ID
}

// ---------------------------------------------------------------------------
// Tests for getBucketSumAmountByBucketID
// ---------------------------------------------------------------------------

// TestGetBucketSumAmountByBucketID verifies single-bucket sum queries.
func TestGetBucketSumAmountByBucketID(t *testing.T) {
	resetDB(t)
	require := require.New(t)

	// No rows → zero, no error.
	got, err := getBucketSumAmountByBucketID(testGormDB, 999)
	require.NoError(err)
	require.Equal(decimal.Zero.String(), got.String(), "missing bucket should return 0")

	// Bucket 1: +500, +300 → 800.
	insertVote(t, AccountVote{BlockHeight: 1, BucketID: 1, Address: "addrA", ActType: "StakeCreate", Amount: decimal.NewFromInt(500)})
	insertVote(t, AccountVote{BlockHeight: 2, BucketID: 1, Address: "addrA", ActType: "DepositToStake", Amount: decimal.NewFromInt(300)})

	got, err = getBucketSumAmountByBucketID(testGormDB, 1)
	require.NoError(err)
	require.Equal(decimal.NewFromInt(800).String(), got.String(), "bucket 1 sum should be 800")

	// Bucket 2: +1000, -200 → 800.
	insertVote(t, AccountVote{BlockHeight: 1, BucketID: 2, Address: "addrB", ActType: "StakeCreate", Amount: decimal.NewFromInt(1000)})
	insertVote(t, AccountVote{BlockHeight: 2, BucketID: 2, Address: "addrB", ActType: "Unstake", Amount: decimal.NewFromInt(-200)})

	got, err = getBucketSumAmountByBucketID(testGormDB, 2)
	require.NoError(err)
	require.Equal(decimal.NewFromInt(800).String(), got.String(), "bucket 2 sum should be 800")
}

// ---------------------------------------------------------------------------
// Tests for getBucketInfoByBucketID
// ---------------------------------------------------------------------------

// TestGetBucketInfoByBucketID verifies that the most-recent row (highest id) is returned.
func TestGetBucketInfoByBucketID(t *testing.T) {
	resetDB(t)
	require := require.New(t)

	// Bucket 1: create then candidate change — latest row (ChangeCandidate) should win.
	insertVote(t, AccountVote{BlockHeight: 1, BucketID: 1, Address: "addrA", Candidate: "cand1", ActType: "StakeCreate", AutoStake: true, Duration: 7})
	insertVote(t, AccountVote{BlockHeight: 2, BucketID: 1, Address: "addrA", Candidate: "cand2", ActType: "ChangeCandidate", AutoStake: true, Duration: 7})

	info, err := getBucketInfoByBucketID(testGormDB, 1)
	require.NoError(err)
	require.NotNil(info)
	require.Equal("addrA", info.Address)
	require.Equal("cand2", info.Candidate, "must return latest row (ChangeCandidate)")
	require.True(info.AutoStake)
	require.Equal(uint32(7), info.Duration)

	// Bucket 2: single row.
	insertVote(t, AccountVote{BlockHeight: 1, BucketID: 2, Address: "addrB", Candidate: "cand3", ActType: "StakeCreate", AutoStake: false, Duration: 14})

	info2, err := getBucketInfoByBucketID(testGormDB, 2)
	require.NoError(err)
	require.NotNil(info2)
	require.Equal("addrB", info2.Address)
	require.Equal("cand3", info2.Candidate)
	require.False(info2.AutoStake)
	require.Equal(uint32(14), info2.Duration)
}

// ---------------------------------------------------------------------------
// Tests for getBucketIDsByAddressWithHeight
// ---------------------------------------------------------------------------

// TestGetBucketIDsByAddressWithHeight_Basic verifies that buckets currently
// owned by addr are returned, and buckets whose latest row at height points to
// a different address are excluded.
func TestGetBucketIDsByAddressWithHeight_Basic(t *testing.T) {
	resetDB(t)
	require := require.New(t)

	addrA := "io1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqd39ym7"
	addrB := "io1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqfvnh97"

	// Bucket 10: owned by addrA only.
	insertVote(t, AccountVote{BlockHeight: 1, BucketID: 10, Address: addrA, ActType: "StakeCreate"})

	// Bucket 20: owned by addrA only.
	insertVote(t, AccountVote{BlockHeight: 1, BucketID: 20, Address: addrA, ActType: "StakeCreate"})

	// Bucket 30: owned by addrB — must NOT appear in addrA's result.
	insertVote(t, AccountVote{BlockHeight: 1, BucketID: 30, Address: addrB, ActType: "StakeCreate"})

	ids, err := getBucketIDsByAddressWithHeight(testGormDB, addrA, 5)
	require.NoError(err)
	require.ElementsMatch([]uint64{10, 20}, ids)

	idsB, err := getBucketIDsByAddressWithHeight(testGormDB, addrB, 5)
	require.NoError(err)
	require.ElementsMatch([]uint64{30}, idsB)
}

// TestGetBucketIDsByAddressWithHeight_OwnershipTransfer verifies the most
// important invariant: after a TransferStake, the bucket no longer appears in
// the original owner's result but does appear in the new owner's result.
// This is precisely what the 3-step SQL must handle correctly.
func TestGetBucketIDsByAddressWithHeight_OwnershipTransfer(t *testing.T) {
	resetDB(t)
	require := require.New(t)

	addrA := "io1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqd39ym7"
	addrB := "io1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqfvnh97"

	// Height 1: addrA creates bucket 10.
	insertVote(t, AccountVote{BlockHeight: 1, BucketID: 10, Address: addrA, ActType: "StakeCreate", Amount: decimal.NewFromInt(1000)})
	// Height 1: addrA creates bucket 20 as well.
	insertVote(t, AccountVote{BlockHeight: 1, BucketID: 20, Address: addrA, ActType: "StakeCreate", Amount: decimal.NewFromInt(500)})
	// Height 3: bucket 10 transferred — addrA loses it, addrB gains it.
	insertVote(t, AccountVote{BlockHeight: 3, BucketID: 10, Address: addrA, ActType: "TransferStake", Amount: decimal.NewFromInt(-1000)})
	insertVote(t, AccountVote{BlockHeight: 3, BucketID: 10, Address: addrB, ActType: "TransferStake", Amount: decimal.NewFromInt(1000)})

	// At height 2 (before transfer): addrA owns bucket 10 and 20.
	idsA2, err := getBucketIDsByAddressWithHeight(testGormDB, addrA, 2)
	require.NoError(err)
	require.ElementsMatch([]uint64{10, 20}, idsA2, "before transfer addrA should own both buckets")

	// At height 4 (after transfer):
	//   addrA retains bucket 20, loses bucket 10.
	//   addrB gains bucket 10.
	idsA4, err := getBucketIDsByAddressWithHeight(testGormDB, addrA, 4)
	require.NoError(err)
	require.ElementsMatch([]uint64{20}, idsA4, "after transfer addrA must lose bucket 10")

	idsB4, err := getBucketIDsByAddressWithHeight(testGormDB, addrB, 4)
	require.NoError(err)
	require.ElementsMatch([]uint64{10}, idsB4, "after transfer addrB must own bucket 10")
}

// TestGetBucketIDsByAddressWithHeight_HeightCap verifies that rows above the
// given height are invisible (the height cap is strictly enforced).
func TestGetBucketIDsByAddressWithHeight_HeightCap(t *testing.T) {
	resetDB(t)
	require := require.New(t)

	addrA := "io1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqd39ym7"
	addrB := "io1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqfvnh97"

	insertVote(t, AccountVote{BlockHeight: 1, BucketID: 10, Address: addrA, ActType: "StakeCreate"})
	// Transfer at height 10 — only visible when height >= 10.
	insertVote(t, AccountVote{BlockHeight: 10, BucketID: 10, Address: addrA, ActType: "TransferStake", Amount: decimal.NewFromInt(-1)})
	insertVote(t, AccountVote{BlockHeight: 10, BucketID: 10, Address: addrB, ActType: "TransferStake", Amount: decimal.NewFromInt(1)})

	// At height 9: transfer not yet visible → addrA owns bucket 10.
	ids9, err := getBucketIDsByAddressWithHeight(testGormDB, addrA, 9)
	require.NoError(err)
	require.ElementsMatch([]uint64{10}, ids9, "before height 10, addrA still owns bucket 10")

	// At height 10: transfer visible → addrA no longer owns it.
	ids10, err := getBucketIDsByAddressWithHeight(testGormDB, addrA, 10)
	require.NoError(err)
	require.Empty(ids10, "at height 10, addrA no longer owns bucket 10")

	idsB10, err := getBucketIDsByAddressWithHeight(testGormDB, addrB, 10)
	require.NoError(err)
	require.ElementsMatch([]uint64{10}, idsB10)
}

// ---------------------------------------------------------------------------
// Tests for getForwardToAddressByFrom
// ---------------------------------------------------------------------------

// TestGetForwardToAddressByFrom_Found verifies that the most recent
// GovernaceForward forward_to value is returned for the given address.
func TestGetForwardToAddressByFrom_Found(t *testing.T) {
	resetDB(t)
	require := require.New(t)

	addrA := "io1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqd39ym7"
	addrB := "io1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqfvnh97"
	addrC := "io1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq8w6mvt"

	// Two GovernaceForward entries — the later one (addrC) should win.
	insertVote(t, AccountVote{BlockHeight: 1, BucketID: 1, Address: addrA, ForwardTo: addrB, ActType: "GovernaceForward"})
	insertVote(t, AccountVote{BlockHeight: 2, BucketID: 2, Address: addrA, ForwardTo: addrC, ActType: "GovernaceForward"})

	// An unrelated row that must NOT interfere (different act_type, empty ForwardTo).
	insertVote(t, AccountVote{BlockHeight: 3, BucketID: 3, Address: addrA, ForwardTo: "", ActType: "StakeCreate"})

	to, err := getForwardToAddressByFrom(testGormDB, addrA)
	require.NoError(err)
	require.Equal(addrC, to, "should return the most recent GovernaceForward forward_to")
}

// TestGetForwardToAddressByFrom_NotFound verifies that an empty string is
// returned when no GovernaceForward row exists for the given address.
func TestGetForwardToAddressByFrom_NotFound(t *testing.T) {
	resetDB(t)
	require := require.New(t)

	addrA := "io1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqd39ym7"
	// Only a StakeCreate row, no GovernaceForward.
	insertVote(t, AccountVote{BlockHeight: 1, BucketID: 1, Address: addrA, ActType: "StakeCreate"})

	to, err := getForwardToAddressByFrom(testGormDB, addrA)
	require.NoError(err)
	require.Empty(to)
}
