package models

import (
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testCandidateDB *gorm.DB

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V16).
			Port(9991),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("WARNING: embedded postgres failed to start: %v\n  candidate fetch tests will be skipped.\n", err)
		os.Exit(m.Run())
	}

	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=9991 sslmode=disable"
	gormLogger := logger.New(
		log.New(os.Stdout, "[gorm] ", log.LstdFlags),
		logger.Config{
			SlowThreshold: 200 * time.Millisecond,
			LogLevel:      logger.Info,
			Colorful:      false,
		},
	)
	var err error
	testCandidateDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   gormLogger,
	})
	if err != nil {
		fmt.Printf("WARNING: failed to connect to embedded postgres: %v\n", err)
		code := m.Run()
		_ = pg.Stop()
		os.Exit(code)
	}

	if err := testCandidateDB.AutoMigrate(&Candidate{}); err != nil {
		fmt.Printf("WARNING: AutoMigrate Candidate failed: %v\n", err)
		testCandidateDB = nil
	}

	code := m.Run()
	_ = pg.Stop()
	os.Exit(code)
}

func resetCandidate(t *testing.T) {
	t.Helper()
	if testCandidateDB == nil {
		t.Skip("embedded postgres not available")
	}
	require.NoError(t, testCandidateDB.Exec("TRUNCATE candidate RESTART IDENTITY CASCADE").Error)
}

// seedTestnetRow inserts a row that mimics the production candidate row on
// hz-bm3-db testnet (id=50, block_height=36844340) for owner
// io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02 — the row the candidate plugin
// was failing to find at block 43063796.
func seedTestnetRow(t *testing.T) {
	t.Helper()
	require.NoError(t, testCandidateDB.Create(&Candidate{
		BlockHeight:     36844340,
		Name:            "test111",
		OwnerAddress:    "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02",
		CandidateID:     "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02",
		OperatorAddress: "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02",
		ActType:         "CandidateUpdate",
	}).Error)
}

// TestFetchByOwnerAddressWithHeight_FixedIdiom verifies that the current
// (post-fix) implementation finds the seeded row when called inside a tx —
// the production failure mode that motivated the fix.
func TestFetchByOwnerAddressWithHeight_FixedIdiom(t *testing.T) {
	resetCandidate(t)
	seedTestnetRow(t)

	err := testCandidateDB.Transaction(func(tx *gorm.DB) error {
		c := &Candidate{}
		if err := c.FetchByOwnerAddressWithHeight(
			"io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02",
			43063796,
			tx,
		); err != nil {
			return err
		}
		require.Equal(t, "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02", c.OwnerAddress)
		require.Equal(t, "test111", c.Name)
		require.EqualValues(t, 36844340, c.BlockHeight)
		return nil
	})
	require.NoError(t, err, "fixed idiom must find the seeded row inside tx")
}

// TestFetchByOwnerOrOperatorWithHeight_OperatorMatch covers the bug that
// motivated this fix: at testnet block 43063855, the CandidateUpdate is
// signed by an address that is the candidate's *operator* (just set at
// block 43063850), not its owner. The pre-fix code looked up only by
// owner and returned ErrRecordNotFound; the new helper falls back to
// operator and finds the row.
func TestFetchByOwnerOrOperatorWithHeight_OperatorMatch(t *testing.T) {
	resetCandidate(t)

	owner := "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02"
	newOperator := "io17qs77wkufe66gv7r2m9gq3naxr64zxat0yjvzp"

	// Earlier CandidateRegister/Update where operator = owner.
	require.NoError(t, testCandidateDB.Create(&Candidate{
		BlockHeight:     36844340,
		Name:            "test111",
		OwnerAddress:    owner,
		OperatorAddress: owner,
		CandidateID:     owner,
		ActType:         "CandidateUpdate",
	}).Error)
	// Later CandidateUpdate that switches operator to a fresh address.
	require.NoError(t, testCandidateDB.Create(&Candidate{
		BlockHeight:     43063850,
		Name:            "bot1",
		OwnerAddress:    owner,
		OperatorAddress: newOperator,
		CandidateID:     owner,
		ActType:         "CandidateUpdate",
	}).Error)

	// The new operator now signs a CandidateUpdate; we must find the
	// candidate via its operator field.
	err := testCandidateDB.Transaction(func(tx *gorm.DB) error {
		c := &Candidate{}
		if err := c.FetchByOwnerOrOperatorWithHeight(newOperator, 43063855, tx); err != nil {
			return err
		}
		require.Equal(t, owner, c.OwnerAddress, "owner stays as the registrant")
		require.Equal(t, newOperator, c.OperatorAddress, "operator is the new one")
		require.EqualValues(t, 43063850, c.BlockHeight, "should return the most recent row")
		return nil
	})
	require.NoError(t, err)
}

// TestFetchByOwnerOrOperatorWithHeight_OwnerMatch verifies the simple
// case still works: when the sender is the owner, the helper finds the
// row via the owner_address field.
func TestFetchByOwnerOrOperatorWithHeight_OwnerMatch(t *testing.T) {
	resetCandidate(t)
	seedTestnetRow(t)

	err := testCandidateDB.Transaction(func(tx *gorm.DB) error {
		c := &Candidate{}
		return c.FetchByOwnerOrOperatorWithHeight(
			"io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02",
			43063796,
			tx,
		)
	})
	require.NoError(t, err)
}

// TestFetchByOwnerOrOperatorWithHeight_NotFound verifies that when neither
// owner nor operator matches, ErrRecordNotFound is returned.
func TestFetchByOwnerOrOperatorWithHeight_NotFound(t *testing.T) {
	resetCandidate(t)
	seedTestnetRow(t)

	err := testCandidateDB.Transaction(func(tx *gorm.DB) error {
		c := &Candidate{}
		return c.FetchByOwnerOrOperatorWithHeight(
			"io1unknownaddressunknownaddressunknownaddrk",
			43063796,
			tx,
		)
	})
	require.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

// TestFetchByCandidateIDWithHeight_FixedIdiom mirrors the FixedIdiom test
// for the by-candidate-id variant (which the plugin uses for
// CandidateActivate and CandidateEndorsement-Revoke handlers).
func TestFetchByCandidateIDWithHeight_FixedIdiom(t *testing.T) {
	resetCandidate(t)
	seedTestnetRow(t)

	err := testCandidateDB.Transaction(func(tx *gorm.DB) error {
		c := &Candidate{}
		if err := c.FetchByCandidateIDWithHeight(
			"io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02",
			43063796,
			tx,
		); err != nil {
			return err
		}
		require.Equal(t, "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02", c.CandidateID)
		return nil
	})
	require.NoError(t, err)
}

// TestFetchByOwnerAddressWithHeight_HeightCap verifies the height filter
// is enforced (a row at height H is invisible when fetching at height < H).
func TestFetchByOwnerAddressWithHeight_HeightCap(t *testing.T) {
	resetCandidate(t)
	seedTestnetRow(t) // height = 36844340

	err := testCandidateDB.Transaction(func(tx *gorm.DB) error {
		c := &Candidate{}
		return c.FetchByOwnerAddressWithHeight(
			"io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02",
			36844339, // one below the seed
			tx,
		)
	})
	require.True(t, errors.Is(err, gorm.ErrRecordNotFound),
		"row at height 36844340 must be invisible when query caps at 36844339; got %v", err)
}

// TestFetchByOwnerAddressWithHeight_LatestRow verifies that when multiple
// rows share an owner_address, the highest-block-height (then highest-id)
// row is returned — the candidate plugin relies on this for "latest state
// at height H".
func TestFetchByOwnerAddressWithHeight_LatestRow(t *testing.T) {
	resetCandidate(t)
	owner := "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02"

	// Seed three rows at increasing heights.
	for _, h := range []uint64{29405739, 36650562, 36844340} {
		require.NoError(t, testCandidateDB.Create(&Candidate{
			BlockHeight:  h,
			OwnerAddress: owner,
			CandidateID:  owner,
			Name:         "test111",
			ActType:      "CandidateUpdate",
		}).Error)
	}

	err := testCandidateDB.Transaction(func(tx *gorm.DB) error {
		c := &Candidate{}
		if err := c.FetchByOwnerAddressWithHeight(owner, 43063796, tx); err != nil {
			return err
		}
		require.EqualValues(t, 36844340, c.BlockHeight, "should return the most recent row")
		return nil
	})
	require.NoError(t, err)
}
