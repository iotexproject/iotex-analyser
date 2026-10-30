package db

import (
	"fmt"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	// A runtime dir of its own: the default one is shared, and go test runs the
	// packages that use embedded postgres in parallel.
	runtime, err := os.MkdirTemp("", "embedded-postgres-")
	if err != nil {
		fmt.Printf("WARNING: %v\n  DB tests will be skipped.\n", err)
		return m.Run()
	}
	defer os.RemoveAll(runtime)
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().Port(9996).RuntimePath(runtime),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("WARNING: embedded postgres failed to start: %v\n  DB tests will be skipped.\n", err)
		return m.Run()
	}
	defer pg.Stop()

	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=9996 sslmode=disable"
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err == nil {
		err = gormDB.AutoMigrate(&IndexHeight{})
	}
	if err != nil {
		fmt.Printf("WARNING: embedded postgres not usable: %v\n  DB tests will be skipped.\n", err)
		return m.Run()
	}
	SetDB(gormDB)
	return m.Run()
}

// UpdateIndexHeightByTx caches a height before its transaction commits, and keeps it
// after a rollback. GetCommittedIndexHeight reads what is committed.
func TestGetCommittedIndexHeight(t *testing.T) {
	if db == nil {
		t.Skip("embedded postgres not available")
	}
	r := require.New(t)
	r.NoError(db.Exec("TRUNCATE index_heights").Error)
	ClearIndexCache()
	_, err := GetIndexHeight("v2") // creates the row at 0
	r.NoError(err)
	r.NoError(UpdateIndexHeight("v2", 99))

	tx := db.Begin()
	r.NoError(UpdateIndexHeightByTx(tx, "v2", 100))
	cached, err := GetIndexHeight("v2")
	r.NoError(err)
	r.EqualValues(100, cached, "the cache runs ahead of the open transaction")
	committed, err := GetCommittedIndexHeight("v2")
	r.NoError(err)
	r.EqualValues(99, committed)

	r.NoError(tx.Rollback().Error)
	cached, _ = GetIndexHeight("v2")
	r.EqualValues(100, cached, "and keeps the rolled-back height")
	committed, err = GetCommittedIndexHeight("v2")
	r.NoError(err)
	r.EqualValues(99, committed)

	r.NoError(db.Transaction(func(tx *gorm.DB) error {
		return UpdateIndexHeightByTx(tx, "v2", 100)
	}))
	committed, err = GetCommittedIndexHeight("v2")
	r.NoError(err)
	r.EqualValues(100, committed)

	// unlike GetIndexHeight, a missing name is 0 and creates no row
	committed, err = GetCommittedIndexHeight("not-installed")
	r.NoError(err)
	r.Zero(committed)
	var n int64
	r.NoError(db.Model(&IndexHeight{}).Where("name = ?", "not-installed").Count(&n).Error)
	r.Zero(n)
}
