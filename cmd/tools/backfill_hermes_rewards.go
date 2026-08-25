package tools

import (
	"errors"
	"fmt"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/service/hermes_reward"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

// BackfillHermesRewards re-runs the hermes_account_rewards rollup for a range
// of already-indexed epochs.
//
// The plugin only rebuilds an epoch as the indexer crosses its boundary, and
// the indexer never revisits a block, so an epoch skipped at that moment stays
// empty forever -- and Hermes refuses to distribute a batch containing one
// ("bookkeeping info doesn't exist for Epoch N"). This is the way to fill such
// a hole after the fact.
var BackfillHermesRewards = &cli.Command{
	Name:  "backfill-hermes-rewards",
	Usage: "Rebuild hermes_account_rewards for a closed epoch range",
	Flags: []cli.Flag{
		&cli.Uint64Flag{Name: "start", Required: true, Usage: "first epoch to rebuild (inclusive)"},
		&cli.Uint64Flag{Name: "end", Required: true, Usage: "last epoch to rebuild (inclusive)"},
		&cli.BoolFlag{Name: "dry-run", Usage: "compute each epoch in a transaction and roll it back, reporting what would be written"},
		&cli.BoolFlag{Name: "only-missing", Usage: "leave epochs that already have rows untouched"},
		&cli.BoolFlag{Name: "show-rows", Usage: "print every delegate row instead of a per-epoch summary"},
	},
	Action: backfillHermesRewards,
}

// errDryRun is returned from the transaction body to force a rollback after
// the rebuild has been computed and read back. It never escapes to the caller.
var errDryRun = errors.New("dry-run rollback")

type epochOutcome struct {
	Rows            []models.HermesAccountReward
	BlockReward     decimal.Decimal
	EpochReward     decimal.Decimal
	FoundationBonus decimal.Decimal
}

func backfillHermesRewards(c *cli.Context) error {
	start, end := c.Uint64("start"), c.Uint64("end")
	if start == 0 {
		return fmt.Errorf("--start must be greater than 0")
	}
	if end < start {
		return fmt.Errorf("empty epoch range: --start %d is after --end %d", start, end)
	}

	gdb, err := db.Connect()
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}

	dryRun := c.Bool("dry-run")
	onlyMissing := c.Bool("only-missing")
	showRows := c.Bool("show-rows")

	if dryRun {
		fmt.Printf("DRY RUN: epochs %d-%d are rebuilt inside a transaction and rolled back; nothing is committed.\n\n", start, end)
	}

	var rebuilt, empty, skipped, untouched int
	grand := epochOutcome{}
	for epoch := start; epoch <= end; epoch++ {
		before, err := countRewardRows(gdb, epoch)
		if err != nil {
			return err
		}
		if onlyMissing && before > 0 {
			untouched++
			continue
		}

		cov, err := hermes_reward.CheckEpochCoverage(gdb, epoch)
		if err != nil {
			return fmt.Errorf("check coverage for epoch %d: %w", epoch, err)
		}
		if !cov.OK {
			fmt.Printf("epoch %d: skipped -- %s (block_meta rows=%d earliest=%d, epoch starts at %d)\n",
				epoch, cov.Reason, cov.Blocks, cov.Earliest, cov.EpochStart)
			skipped++
			continue
		}

		out, err := rebuildEpoch(gdb, epoch, dryRun)
		if err != nil {
			return err
		}

		if len(out.Rows) == 0 {
			// Coverage was fine, so this is the chain having granted nothing in
			// the epoch, or hermes_voting_results not being populated for it.
			// Either way Hermes will still trip over this epoch -- say so
			// rather than reporting a successful rebuild.
			fmt.Printf("epoch %d: produces 0 rows -- no block_rewards or no hermes_voting_results for it\n", epoch)
			empty++
			continue
		}

		verb := "rebuilt"
		if dryRun {
			verb = "would write"
		}
		fmt.Printf("epoch %d: %s %d delegates  block=%s  epoch=%s  foundation=%s IOTX  (had %d rows)\n",
			epoch, verb, len(out.Rows),
			iotx(out.BlockReward), iotx(out.EpochReward), iotx(out.FoundationBonus), before)
		if showRows {
			for _, row := range out.Rows {
				fmt.Printf("    %-14s block=%-16s epoch=%-16s foundation=%s\n",
					row.CandidateName, iotx(row.BlockReward), iotx(row.EpochReward), iotx(row.FoundationBonus))
			}
		}

		grand.BlockReward = grand.BlockReward.Add(out.BlockReward)
		grand.EpochReward = grand.EpochReward.Add(out.EpochReward)
		grand.FoundationBonus = grand.FoundationBonus.Add(out.FoundationBonus)
		rebuilt++
	}

	fmt.Printf("\ndone: %d epochs with rows, %d producing no rows, %d skipped for coverage, %d left untouched\n",
		rebuilt, empty, skipped, untouched)
	fmt.Printf("totals across %d epochs: block=%s  epoch=%s  foundation=%s IOTX\n",
		rebuilt, iotx(grand.BlockReward), iotx(grand.EpochReward), iotx(grand.FoundationBonus))
	if empty > 0 || skipped > 0 {
		fmt.Printf("warning: %d epoch(s) in %d-%d would still have no rows and will keep blocking Hermes\n",
			empty+skipped, start, end)
	}
	return nil
}

// rebuildEpoch runs the rollup for one epoch and reads the resulting rows back
// inside the same transaction. When dry is true the transaction is rolled back
// afterwards, so the rows describe what a real run would write without any of
// it being committed.
func rebuildEpoch(gdb *gorm.DB, epoch uint64, dry bool) (*epochOutcome, error) {
	out := &epochOutcome{}
	err := gdb.Transaction(func(tx *gorm.DB) error {
		if err := hermes_reward.RebuildAccountRewardTable(tx, epoch); err != nil {
			return err
		}
		if err := tx.Where("epoch_number = ?", epoch).
			Order("candidate_name").
			Find(&out.Rows).Error; err != nil {
			return err
		}
		for _, row := range out.Rows {
			out.BlockReward = out.BlockReward.Add(row.BlockReward)
			out.EpochReward = out.EpochReward.Add(row.EpochReward)
			out.FoundationBonus = out.FoundationBonus.Add(row.FoundationBonus)
		}
		if dry {
			return errDryRun
		}
		return nil
	})
	if err != nil && !errors.Is(err, errDryRun) {
		return nil, fmt.Errorf("rebuild epoch %d: %w", epoch, err)
	}
	return out, nil
}

// iotx renders a Rau amount (1e18 per IOTX) for human consumption.
func iotx(d decimal.Decimal) string {
	return d.Shift(-18).StringFixed(6)
}

func countRewardRows(gdb *gorm.DB, epoch uint64) (int64, error) {
	var n int64
	if err := gdb.Model(&models.HermesAccountReward{}).
		Where("epoch_number = ?", epoch).
		Count(&n).Error; err != nil {
		return 0, fmt.Errorf("count hermes_account_rewards for epoch %d: %w", epoch, err)
	}
	return n, nil
}
