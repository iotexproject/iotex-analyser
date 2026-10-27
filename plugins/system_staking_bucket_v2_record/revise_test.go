package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var testGormDB *gorm.DB

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
		embeddedpostgres.DefaultConfig().Port(9995).RuntimePath(runtime),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("WARNING: embedded postgres failed to start: %v\n  DB tests will be skipped.\n", err)
		return m.Run()
	}
	defer pg.Stop()

	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=9995 sslmode=disable"
	testGormDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err == nil {
		err = testGormDB.AutoMigrate(&models.SystemStakingBucketV2Record{})
	}
	if err != nil {
		fmt.Printf("WARNING: embedded postgres not usable: %v\n  DB tests will be skipped.\n", err)
		testGormDB = nil
	}
	return m.Run()
}

// At the mute height (the block-interval halving), revise restates every still-staked
// bucket with its duration in seconds. The restatement must not move any bucket's
// balance (sum(amount)), and buckets that are already closed must not get one.
func TestReviseAtHalving(t *testing.T) {
	if testGormDB == nil {
		t.Skip("embedded postgres not available")
	}
	r := require.New(t)
	r.NoError(testGormDB.Exec("TRUNCATE system_staking_buckets_v2_record RESTART IDENTITY").Error)

	const halving = 1000
	zero := decimal.Zero
	iotx := func(n int64) decimal.Decimal { return decimal.New(n, 18) }
	row := func(bucket uint64, event string, amount, staked decimal.Decimal) models.SystemStakingBucketV2Record {
		return models.SystemStakingBucketV2Record{
			SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
				BlockHeight:  900,
				BucketID:     bucket,
				EventType:    event,
				ActHash:      fmt.Sprintf("%d-%s", bucket, event),
				Amount:       amount,
				StakedAmount: staked,
				VotingPower:  staked,
				AutoStake:    true,
				Duration:     91,
				DurationType: 0,
				Final:        true,
			},
		}
	}
	rows := []models.SystemStakingBucketV2Record{
		row(1, "Transfer", zero, zero),
		row(1, "Staked", iotx(100), iotx(100)),
		row(2, "Staked", iotx(50), iotx(50)),
		row(2, "Unstaked", iotx(-50), zero),
		row(3, "Staked", iotx(30), iotx(30)),
		row(3, "Unstaked", iotx(-30), zero),
		row(3, "Withdrawal", zero, zero),
		row(4, "Staked", iotx(10), iotx(10)),
		row(4, "Merged", iotx(-10), zero), // merged into bucket 1
		row(1, "Merged", iotx(10), iotx(110)),
		row(5, "Staked", iotx(20), iotx(20)),
		row(5, "DelegateChanged", zero, iotx(20)),
	}
	r.NoError(testGormDB.Create(&rows).Error)

	balances := func() map[uint64]string {
		var res []struct {
			BucketID uint64
			Sum      string
		}
		r.NoError(testGormDB.Raw("SELECT bucket_id, sum(amount)::text AS sum FROM system_staking_buckets_v2_record GROUP BY bucket_id").Scan(&res).Error)
		m := map[uint64]string{}
		for _, x := range res {
			m[x.BucketID] = x.Sum
		}
		return m
	}
	before := balances()

	p := systemStakingBucketPlugin{muteHeight: halving}
	r.NoError(testGormDB.Transaction(func(tx *gorm.DB) error {
		return p.revise(context.Background(), tx)
	}))

	r.Equal(before, balances())

	var copies []models.SystemStakingBucketV2Record
	r.NoError(testGormDB.Where("block_height = ?", halving).Order("bucket_id").Find(&copies).Error)
	r.Len(copies, 2, "only the open buckets 1 and 5 are restated")
	for i, want := range []struct {
		bucket uint64
		event  string
		staked decimal.Decimal
	}{
		{1, "Merged", iotx(110)},
		{5, "DelegateChanged", iotx(20)},
	} {
		c := copies[i]
		r.Equal(want.bucket, c.BucketID)
		r.Equal(want.event, c.EventType)
		r.True(c.Amount.IsZero(), "bucket %d: copy carries amount %s", c.BucketID, c.Amount)
		r.True(want.staked.Equal(c.StakedAmount), "bucket %d: staked %s", c.BucketID, c.StakedAmount)
		r.EqualValues(1, c.DurationType)
		r.EqualValues(91*86400/2, c.Duration)
	}
}
