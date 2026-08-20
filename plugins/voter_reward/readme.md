---
plugin: voter_reward
---

## Introduction

Indexes the IIP-59 on-chain voter reward distribution introduced by the
Zanzibar hardfork. Four tables, one plugin, one index height — the era summary
has to agree with the payout rows it summarises, which independent index
heights cannot guarantee.

| table | grain | source |
|---|---|---|
| `voter_reward_distribution` | one voter's share in one chunk | `DelegateVoterRewardsDistributed` receipt event, parallel arrays flattened |
| `voter_reward_era` | one settlement | `CURSOR_PROGRESS` / `EPOCH_DRAIN_OVERRUN` reward logs + the `VoterRewardDistribution` read state |
| `delegate_reward_config` | one delegate per era | `DelegatePayoutAddress` + `DelegateRewardSnapshot` read state, plus `VoterRewardOptInSet` events for attribution |
| `voter_reward_destination` | one change | `VoterRewardDestinationSet` receipt event |

## Two things that are easy to get wrong

**`compounded[i]`, not `compoundBucketIds[i] == 0`, decides whether a share was
redeposited.** Native bucket index 0 is a real bucket, which is why the event
carries a parallel bool array at all.

**Opt-in cannot be indexed from events alone.** The fork-block Hermes migration
flips the opt-in bit inside `CreatePreStates` and emits no action and no log.
This plugin therefore reconciles every candidate's routing from chain state
once per era; `opt_in_source` records whether an explicit
`SetVoterRewardOptIn` was observed (`action`) or not (`migration`).

## Consistency check

For any era, `Σ voter_reward_distribution.amount` must be `≤ Σ` the frozen
per-delegate allocations (`voter_reward_era.total_frozen`). The protocol's
payout clamp guarantees the inequality; a violation means the decode is wrong,
not that the chain overpaid. Any shortfall is integer-division dust or a
self-stake predicate skew, and rolls into a later era.

## Re-indexing

Reset the plugin's index height to 0 and restart. `AutoMigrate` drops and
rebuilds the four tables at height 0, which is what makes this safe.

Do **not** re-run an already-indexed range by rewinding the index height to
some middle value. `voter_reward_distribution` carries a unique index on
`(action_hash, log_index, row_index)` so the payout rows would be deduplicated,
but `voter_reward_era` accumulates `voter_count` and `total_distributed` as
chunks land — a partial re-run inflates them, and the inflation is not
detectable from the table afterwards.
