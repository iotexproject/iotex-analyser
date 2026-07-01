package main

import (
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Burn-drop sender addresses. These were hardcoded in the legacy
// sync_delegate_rewards() proc; carry them forward verbatim so the produced
// burn_reward numbers match the historical totals.
const (
	burnDropSender1 = "io1h49cfw0zcj63chk6awu08u6xem8kpr350r5p57"
	burnDropSender2 = "io1unvkgm98ma3r2fnfrhep24arjxf6kc8stx0nuc"
)

// upsertSQL is a byte-for-byte port of the legacy sync_delegate_rewards()
// proc from backup/iotex-analytics/sql/hermes_delegate_total_rewards.sql:
// per-candidate SUM of hermes rewards, LEFT JOIN'd against burn-drop
// DepositToStake amounts, upserted by candidate address.
const upsertSQL = `
INSERT INTO delegate_rewards (candidate, block_reward, epoch_reward, foundation_bonus, burn_reward)
SELECT a.candidate,
       COALESCE(a.block_reward, 0),
       COALESCE(a.epoch_reward, 0),
       COALESCE(a.foundation_bonus, 0),
       COALESCE(s.burn_reward, 0)
FROM (
    SELECT d.candidate,
           SUM(h.block_reward)     AS block_reward,
           SUM(h.epoch_reward)     AS epoch_reward,
           SUM(h.foundation_bonus) AS foundation_bonus
    FROM delegate d
    LEFT JOIN hermes_account_rewards h ON h.candidate_name = d.name
    GROUP BY d.candidate
) a
LEFT JOIN (
    SELECT candidate, SUM(amount) AS burn_reward
    FROM staking_actions
    WHERE sender IN (?, ?)
      AND act_type = 'DepositToStake'
    GROUP BY candidate
) s ON a.candidate = s.candidate
ON CONFLICT (candidate) DO UPDATE SET
    block_reward     = EXCLUDED.block_reward,
    epoch_reward     = EXCLUDED.epoch_reward,
    foundation_bonus = EXCLUDED.foundation_bonus,
    burn_reward      = EXCLUDED.burn_reward;
`

// syncDelegateRewards rebuilds the delegate_rewards rollup. Safe to call
// repeatedly; the ON CONFLICT UPDATE makes each run a full snapshot refresh.
// Guarded against running before the source tables exist (cold start after
// a fresh analyser deploy where hermes/staking_actions have not migrated
// yet).
func syncDelegateRewards() error {
	for _, tbl := range []string{"delegate", "hermes_account_rewards", "staking_actions"} {
		var exists bool
		if err := db.DB().Raw(
			"SELECT to_regclass(?) IS NOT NULL", tbl,
		).Scan(&exists).Error; err != nil {
			return errors.Wrapf(err, "failed to probe %s", tbl)
		}
		if !exists {
			log.L().Info("delegate_rewards: source table not yet present, skipping tick",
				zap.String("table", tbl))
			return nil
		}
	}
	return db.DB().Transaction(func(tx *gorm.DB) error {
		return tx.Exec(upsertSQL, burnDropSender1, burnDropSender2).Error
	})
}
