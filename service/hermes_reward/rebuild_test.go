package hermes_reward

import (
	"fmt"
	"math/big"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestContributorsTo covers the filter on its own: no database, so it runs
// everywhere.
func TestContributorsTo(t *testing.T) {
	all := []string{"alpha", "bravo", "charlie"}

	for _, tc := range []struct {
		name    string
		names   []string
		optedIn map[string]bool
		want    []string
	}{{
		name:    "no opt-in data does not filter",
		names:   all,
		optedIn: map[string]bool{},
		want:    all,
	}, {
		// Pre-Zanzibar rebuilds must reproduce the old split exactly, and a
		// nil map is what an epoch before the fork produces.
		name:    "nil opt-in data does not filter",
		names:   all,
		optedIn: nil,
		want:    all,
	}, {
		name:    "opted-in delegates are dropped",
		names:   all,
		optedIn: map[string]bool{"alpha": true, "charlie": true},
		want:    []string{"bravo"},
	}, {
		// A funded pool whose every delegate looks opted in means the opt-in
		// data is a step behind block_rewards. Reproducing the old split beats
		// dropping the epoch's rewards.
		name:    "filtering everything falls back to the full set",
		names:   all,
		optedIn: map[string]bool{"alpha": true, "bravo": true, "charlie": true},
		want:    all,
	}, {
		// A sole delegate is attributed directly by the caller, never split,
		// so there is nothing to filter.
		name:    "single delegate is left alone",
		names:   []string{"alpha"},
		optedIn: map[string]bool{"alpha": true},
		want:    []string{"alpha"},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got := contributorsTo(tc.names, tc.optedIn, "io1vault", 65495)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

// openTestDB connects to a scratch Postgres and gives the test its own schema.
// Skips when TEST_PG_DSN is unset, e.g.
//
//	TEST_PG_DSN='host=127.0.0.1 port=5432 user=postgres dbname=postgres sslmode=disable' go test ./service/hermes_reward/
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	schema := fmt.Sprintf("hermes_reward_test_%d", os.Getpid())
	if err := db.Exec("DROP SCHEMA IF EXISTS " + schema + " CASCADE").Error; err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { db.Exec("DROP SCHEMA IF EXISTS " + schema + " CASCADE") })
	if err := db.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatalf("search_path: %v", err)
	}
	return db
}

// TestOptedInDelegates checks the era the opt-in set is read from, including
// the fallback taken while an epoch is still being indexed live.
func TestOptedInDelegates(t *testing.T) {
	db := openTestDB(t)
	if err := db.Exec(`CREATE TABLE delegate_reward_config (
		delegate_name text NOT NULL, era bigint NOT NULL, opted_in bool NOT NULL)`).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	// Eras run every EpochsPerRewardEra epochs; era N settles [N-24, N-1].
	for _, r := range []struct {
		name    string
		era     uint64
		optedIn bool
	}{
		{"alpha", 65472, true}, {"bravo", 65472, false},
		{"alpha", 65496, true}, {"bravo", 65496, true}, {"charlie", 65496, false},
	} {
		if err := db.Exec(`INSERT INTO delegate_reward_config VALUES (?,?,?)`,
			r.name, r.era, r.optedIn).Error; err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	for _, tc := range []struct {
		name  string
		epoch uint64
		want  []string
	}{
		// 65471 is settled by era 65472: only alpha had opted in by then.
		{"covering era is the smallest one above the epoch", 65471, []string{"alpha"}},
		// 65495 is settled by era 65496, where bravo has joined.
		{"later epoch picks up the later era", 65495, []string{"alpha", "bravo"}},
		// Past the last settled era -- an epoch being indexed live. Falls back
		// to the newest on record rather than returning nothing, which would
		// silently disable the filter.
		{"beyond the last era falls back to the newest", 65500, []string{"alpha", "bravo"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := optedInDelegates(db, tc.epoch)
			if err != nil {
				t.Fatalf("optedInDelegates: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("epoch %d: got %v, want %v", tc.epoch, got, tc.want)
			}
			for _, w := range tc.want {
				if !got[w] {
					t.Fatalf("epoch %d: got %v, want %v", tc.epoch, got, tc.want)
				}
			}
		})
	}

	t.Run("empty table does not filter", func(t *testing.T) {
		if err := db.Exec(`DELETE FROM delegate_reward_config`).Error; err != nil {
			t.Fatalf("delete: %v", err)
		}
		got, err := optedInDelegates(db, 65495)
		if err != nil {
			t.Fatalf("optedInDelegates: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %v, want empty", got)
		}
	})
}

// TestBreakdownRewardsExcludesOptedIn is the end-to-end case: a pool funded by
// one delegate, shared with four that opted in and therefore pay into their
// own owner addresses instead. Shaped after mainnet epochs 65472-65495, where
// one address received 14,842 IOTX entirely from the delegates that had not
// opted in and the split handed most of it to the ones that had.
func TestBreakdownRewardsExcludesOptedIn(t *testing.T) {
	db := openTestDB(t)
	for _, ddl := range []string{
		`CREATE TABLE delegate_reward_config (delegate_name text NOT NULL, era bigint NOT NULL, opted_in bool NOT NULL)`,
		`CREATE TABLE block_meta (epoch_num bigint, producer_name text, producer_address text,
			expected_producer_name text, expected_producer_address text)`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	const epoch = 65495
	funder := "goodwill"
	freeloaders := []string{"alpha", "bravo", "charlie", "delta"}
	all := append([]string{funder}, freeloaders...)

	for _, name := range all {
		optedIn := name != funder
		if err := db.Exec(`INSERT INTO delegate_reward_config VALUES (?,?,?)`, name, 65496, optedIn).Error; err != nil {
			t.Fatalf("insert config: %v", err)
		}
		// Equal production keeps the block-reward split even, so the only
		// thing moving between the two runs is who is in the set.
		for i := 0; i < 10; i++ {
			if err := db.Exec(`INSERT INTO block_meta VALUES (?,?,?,?,?)`,
				epoch, name, "addr_"+name, name, "addr_"+name).Error; err != nil {
				t.Fatalf("insert block_meta: %v", err)
			}
		}
	}

	votes := map[string]*big.Int{}
	for _, name := range all {
		votes[name] = big.NewInt(1_000_000)
	}

	pool := big.NewInt(14_842)
	sum := func(m map[string][]*big.Int) *big.Int {
		total := big.NewInt(0)
		for _, v := range m {
			for _, part := range v {
				total.Add(total, part)
			}
		}
		return total
	}

	// Before: the whole registered set shares the pool.
	before, err := breakdownRewards(db, epoch, all, votes, big.NewInt(0), pool, big.NewInt(0))
	if err != nil {
		t.Fatalf("breakdownRewards (unfiltered): %v", err)
	}
	beforeFunder := before[funder][1]
	if beforeFunder.Cmp(pool) >= 0 {
		t.Fatalf("funder already gets the whole pool unfiltered (%s of %s); fixture does not reproduce the bug",
			beforeFunder, pool)
	}

	// After: only the delegates that still pay in.
	optedIn, err := optedInDelegates(db, epoch)
	if err != nil {
		t.Fatalf("optedInDelegates: %v", err)
	}
	kept := contributorsTo(all, optedIn, "io1vault", epoch)
	if len(kept) != 1 || kept[0] != funder {
		t.Fatalf("kept %v, want [%s]", kept, funder)
	}
	after, err := breakdownRewards(db, epoch, kept, votes, big.NewInt(0), pool, big.NewInt(0))
	if err != nil {
		t.Fatalf("breakdownRewards (filtered): %v", err)
	}
	afterFunder := after[funder][1]
	if afterFunder.Cmp(pool) != 0 {
		t.Fatalf("funder got %s of the %s pool, want all of it", afterFunder, pool)
	}

	// The pool is conserved either way: this moves attribution, not money.
	// Each share is an integer division, so a split across n delegates can
	// leave up to n-1 behind -- the same truncation that shows up on mainnet
	// as ~0.5 rau per voter of dust between total_frozen and total_distributed.
	conserved := func(label string, got *big.Int, recipients int) {
		t.Helper()
		lost := new(big.Int).Sub(pool, got)
		if lost.Sign() < 0 || lost.Cmp(big.NewInt(int64(recipients))) >= 0 {
			t.Fatalf("%s split totals %s of %s, lost %s (truncation allows < %d)",
				label, got, pool, lost, recipients)
		}
	}
	conserved("unfiltered", sum(before), len(all))
	conserved("filtered", sum(after), len(kept))

	t.Logf("funder's share: %s -> %s of a %s pool shared with %d opted-in delegates",
		beforeFunder, afterFunder, pool, len(freeloaders))
}
