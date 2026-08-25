package voter_reward

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// VERSION is bumped whenever the decoded shape changes, so operators know a
// reindex is required.
const VERSION = "1.0.0"

// How a delegate came to be opted in.
const (
	// optInSourceAction means an explicit SetVoterRewardOptIn was indexed.
	optInSourceAction = "action"
	// optInSourceMigration means the delegate was opted in by the fork-block
	// Hermes migration, which emits no action and no receipt log.
	optInSourceMigration = "migration"
)

// Plugin indexes the whole IIP-59 on-chain voter reward surface into four
// tables. They are one plugin rather than four because they have to agree with
// each other: an era summary that is one block ahead of its own payout rows
// reads, in the API, as a settlement that paid less than it did. Four
// independent index heights cannot offer that guarantee.
type Plugin struct {
	Shadow plugin.PluginShadow

	tipHeight  uint64
	candidates *CandidateIndex

	// optInSeen remembers which delegates we watched opt in via an explicit
	// action, so the era reconciliation can tell an action-driven opt-in from
	// the eventless Hermes migration at the fork block.
	optInSeen map[string]uint64

	// eraSeen caches eras already summarised, so the *immutable* half of a
	// settlement is read from the chain once per era rather than once per
	// chunk block.
	//
	// Only eras whose rows are already committed live here; see pendingEraSeen.
	eraSeen map[uint64]bool

	// eraDone remembers settlements already observed complete, so the
	// post-chunk state read stops once there is nothing left to learn.
	eraDone map[uint64]bool

	// chunkEras collects the eras a chunk ran for in the block being
	// processed, so their post-chunk cursor can be re-read once.
	chunkEras map[uint64]struct{}

	pendingRows         []models.VoterRewardDistribution
	pendingDestinations []models.VoterRewardDestination
	pendingEras         map[uint64]*models.VoterRewardEra
	pendingConfigs      []models.DelegateRewardConfig

	// pendingEraSeen and pendingEraDone hold memo entries earned by the batch
	// currently buffered but not yet committed.
	//
	// They exist because the runner replays a batch verbatim when PutBlocks
	// returns an error (server/runner.go does not advance nextHeight). A memo
	// promoted into eraSeen before the transaction lands would make the replay
	// skip ensureEraPlan for an era whose plan was never written — leaving the
	// era row at freeze_height=0/total_frozen=0 forever and its
	// delegate_reward_config rows permanently absent. commit() promotes these
	// only after the transaction succeeds, and drops them if it fails.
	pendingEraSeen map[uint64]bool
	pendingEraDone map[uint64]bool

	// eraStatePass / pendingEraStatePass memoise the state-driven boundary pass
	// separately from eraSeen, which belongs to the cursor path. Sharing one
	// memo let whichever ran first suppress the other; they read different
	// halves of the era and both have to run.
	eraStatePass        map[uint64]bool
	pendingEraStatePass map[uint64]bool
}

// New returns a plugin instance ready to be registered.
//
// It returns a value, not a pointer, because the loader resolves the exported
// `Plugin` symbol with plugin.Lookup, which hands back the *address of the
// package-level variable*. Exporting a `*Plugin` would therefore make the
// symbol a `**Plugin` and fail the Adapter type assertion.
func New(shadow plugin.PluginShadow) Plugin {
	return Plugin{
		Shadow:         shadow,
		candidates:     NewCandidateIndex(),
		optInSeen:      map[string]uint64{},
		eraSeen:        map[uint64]bool{},
		eraDone:        map[uint64]bool{},
		chunkEras:      map[uint64]struct{}{},
		pendingEras:    map[uint64]*models.VoterRewardEra{},
		pendingEraSeen: map[uint64]bool{},
		pendingEraDone: map[uint64]bool{},

		eraStatePass:        map[uint64]bool{},
		pendingEraStatePass: map[uint64]bool{},
	}
}

// table resolves the table name a model writes to, honouring the shadow
// mapping. Every read and write in this plugin goes through it: Start creates
// the shadowed tables, so a write that addressed the model's own TableName
// would target the base table instead.
func (p *Plugin) table(m plugin.Table) string {
	return p.Shadow.ShadowTable(m).TableName()
}

func (p *Plugin) Name() string               { return p.Shadow.ShadowName("voter_reward") }
func (p *Plugin) Version() string            { return VERSION }
func (p *Plugin) Type() plugin.Type          { return plugin.TypeStandard }
func (p *Plugin) BatchSize() int             { return 200 }
func (p *Plugin) Stop(context.Context) error { return nil }

// DependentPlugins keeps candidate metadata available for the denormalised
// delegate name columns.
func (p *Plugin) DependentPlugins() []string { return []string{"candidate"} }

// CatchUpSafe is deliberately not implemented: the tables are a cumulative
// payout ledger, and starting mid-chain would leave permanent holes in a
// voter's earning history rather than merely delaying it.

func (p *Plugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(p.Name(),
		p.Shadow.ShadowTable(models.VoterRewardDistribution{}),
		p.Shadow.ShadowTable(models.VoterRewardEra{}),
		p.Shadow.ShadowTable(models.DelegateRewardConfig{}),
		p.Shadow.ShadowTable(models.VoterRewardDestination{}),
	); err != nil {
		return err
	}
	return p.rehydrateOptIns()
}

// rehydrateOptIns reloads which delegates were seen opting in by an explicit
// action.
//
// Without this the attribution silently degrades on every restart: the era
// reconciliation reads opt-in from chain state, which cannot say *how* a
// delegate opted in, and falls back to "migration" for anything it has no
// action record of. A delegate that sent SetVoterRewardOptIn before the
// restart would then be relabelled as fork-migrated — wrong, and wrong in the
// direction that hides a user action.
func (p *Plugin) rehydrateOptIns() error {
	var rows []models.DelegateRewardConfig
	if err := db.DB().
		Table(p.table(models.DelegateRewardConfig{})).
		Where("opt_in_source = ?", optInSourceAction).
		Select("delegate_id", "opt_in_height").
		Find(&rows).Error; err != nil {
		return errors.Wrap(err, "voter_reward: reload observed opt-in actions")
	}
	for _, row := range rows {
		if _, ok := p.optInSeen[row.DelegateID]; !ok {
			p.optInSeen[row.DelegateID] = row.OptInHeight
		}
	}
	return nil
}

func (p *Plugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := p.putBlock(ctx, blk); err != nil {
		return err
	}
	p.tipHeight = blk.Height()
	return p.commit()
}

func (p *Plugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := p.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	p.tipHeight = blks[len(blks)-1].Height()
	return p.commit()
}

func (p *Plugin) putBlock(ctx context.Context, blk *block.Block) error {
	height := blk.Height()
	epoch := kernel.GetEpochNum(height)
	if err := p.refreshCandidatesIfNeeded(ctx, height, epoch); err != nil {
		return err
	}

	clear(p.chunkEras)
	for _, receipt := range blk.Receipts {
		actHash := hex.EncodeToString(receipt.ActionHash[:])
		for logIndex, l := range receipt.Logs() {
			if err := p.consumeLog(ctx, blk, epoch, actHash, uint32(logIndex), l); err != nil {
				return err
			}
		}
	}
	if err := p.refreshChunkedEras(ctx, height); err != nil {
		return err
	}
	return p.indexFrozenEra(ctx, height, epoch)
}

// refreshChunkedEras re-reads the settlement cursor after a block that ran a
// chunk.
//
// Completion cannot be read off the event stream. The protocol emits
// CURSOR_PROGRESS *before* running a chunk, and refuses to dispatch a chunk
// for an already-complete cursor — so the phase in the log is always the
// pre-drain one and the terminal value never appears there at all. The block
// that finishes a settlement logs the phase it started from, and then nothing
// further is ever emitted for that era. Post-chunk state is therefore the only
// place completion is observable.
func (p *Plugin) refreshChunkedEras(ctx context.Context, height uint64) error {
	if len(p.chunkEras) == 0 {
		return nil
	}
	// Nothing further to learn once a settlement is known complete, and the
	// protocol will not dispatch another chunk for it either.
	pending := false
	for era := range p.chunkEras {
		if !p.isEraDone(era) {
			pending = true
			break
		}
	}
	if !pending {
		return nil
	}
	client := kernel.ChainClient()
	if client == nil {
		return errors.New("voter_reward: no chain client configured")
	}
	state, err := ReadDistributionState(ctx, client, height)
	if err != nil {
		return err
	}
	era := state.GetTargetEra()
	if _, ok := p.chunkEras[era]; !ok {
		// The settlement moved on within the block; the next chunk re-reads.
		return nil
	}
	row := p.era(era)
	row.ScanPhase = state.GetScanPhase()
	row.ResumeVoter = hexOrEmpty(state.GetResumeVoter())
	if state.GetCompleted() || state.GetScanPhase() == ScanPhaseDone {
		if !p.isEraDone(era) {
			p.pendingEraDone[era] = true
			log.L().Info("voter_reward: settlement complete",
				zap.Uint64("era", era), zap.Uint64("height", height))
		}
		row.CompletedHeight = height
		if h := state.GetCompletedHeight(); h != 0 {
			row.CompletedHeight = h
		}
	}
	return nil
}

func (p *Plugin) consumeLog(
	ctx context.Context, blk *block.Block, epoch uint64, actHash string, logIndex uint32, l *action.Log,
) error {
	height := blk.Height()

	rows, err := DecodeDistribution(l)
	if err != nil {
		return err
	}
	for _, row := range rows {
		name := ""
		if info, ok := p.candidates.Lookup(row.Delegate); ok {
			name = info.Name
		}
		p.pendingRows = append(p.pendingRows, models.VoterRewardDistribution{
			BlockHeight:      height,
			EpochNumber:      epoch,
			Era:              row.Era,
			ActionHash:       actHash,
			LogIndex:         logIndex,
			RowIndex:         row.RowIndex,
			DelegateID:       row.Delegate,
			DelegateName:     name,
			VoterAddress:     row.Voter,
			RecipientAddress: row.Recipient,
			Amount:           dec(row.Amount),
			Compounded:       row.Compounded,
			CompoundBucketID: row.CompoundBucketID,
		})
		era := p.era(row.Era)
		era.VoterCount++
		era.TotalDistributed = era.TotalDistributed.Add(dec(row.Amount))
		era.LastChunkAt = height
		if err := p.ensureEraPlan(ctx, row.Era, height); err != nil {
			return err
		}
	}

	if candID, err := DecodeOptIn(l); err != nil {
		return err
	} else if candID != "" {
		// Record the height so the era snapshot can attribute this opt-in to
		// an explicit action rather than the fork-block migration.
		if _, ok := p.optInSeen[candID]; !ok {
			p.optInSeen[candID] = height
		}
		log.L().Info("voter_reward: delegate opted in to on-chain rewards",
			zap.String("delegate", candID), zap.Uint64("height", height))
	}

	if change, err := DecodeDestination(l); err != nil {
		return err
	} else if change != nil {
		p.pendingDestinations = append(p.pendingDestinations, models.VoterRewardDestination{
			BlockHeight:  height,
			ActionHash:   actHash,
			VoterAddress: change.Voter,
			OldRecipient: change.OldRecipient,
			NewRecipient: change.NewRecipient,
		})
	}

	progress, overrun, err := DecodeRewardBookkeeping(l)
	if err != nil {
		return err
	}
	if progress != nil {
		era := p.era(progress.Era)
		// This is the cursor as it stood *before* the chunk ran; the
		// post-chunk value, including completion, is picked up by
		// refreshChunkedEras once the whole block has been read.
		era.ScanPhase = progress.ScanPhase
		era.ResumeVoter = progress.ResumeVoter
		era.LastChunkAt = height
		p.chunkEras[progress.Era] = struct{}{}
		if err := p.ensureEraPlan(ctx, progress.Era, height); err != nil {
			return err
		}
	}
	if overrun != nil {
		era := p.era(overrun.Era)
		era.OverrunResidue = dec(overrun.Residue)
		log.L().Warn("voter_reward: settlement overran its era",
			zap.Uint64("era", overrun.Era),
			zap.Uint64("delegatesRemaining", overrun.DelegatesRemaining),
			zap.String("residue", overrun.Residue.String()),
			zap.Uint64("height", height))
	}
	return nil
}

func (p *Plugin) era(era uint64) *models.VoterRewardEra {
	if row, ok := p.pendingEras[era]; ok {
		return row
	}
	row := &models.VoterRewardEra{
		Era:              era,
		TotalFrozen:      decimal.Zero,
		TotalDistributed: decimal.Zero,
		OverrunResidue:   decimal.Zero,
	}
	p.pendingEras[era] = row
	return row
}

// ensureEraPlan reads the immutable half of a settlement — freeze height,
// per-delegate frozen amounts, delegate count — once per era, and reconciles
// every candidate's opt-in state at the same time.
//
// The reconciliation is a full state sweep rather than an event replay on
// purpose. Opting in arrives by two routes and only one of them emits an
// event: the fork-block Hermes migration flips the bit inside CreatePreStates
// with no action and no log at all. Reading the state instead of the events
// also makes the table self-healing across a re-index or a missed block.
func (p *Plugin) ensureEraPlan(ctx context.Context, era uint64, height uint64) error {
	if p.isEraSeen(era) {
		return nil
	}
	client := kernel.ChainClient()
	if client == nil {
		return errors.New("voter_reward: no chain client configured")
	}
	state, err := ReadDistributionState(ctx, client, height)
	if err != nil {
		return err
	}
	if state.GetTargetEra() != era {
		// The settlement moved on before we read it. Leave the era unmarked so
		// the next chunk retries rather than persisting a half-filled plan.
		log.L().Debug("voter_reward: distribution state is for another era",
			zap.Uint64("want", era), zap.Uint64("got", state.GetTargetEra()))
		return nil
	}

	row := p.era(era)
	row.FreezeHeight = state.GetFreezeHeight()
	row.ScanPhase = state.GetScanPhase()
	if row.FirstChunkAt == 0 {
		row.FirstChunkAt = height
	}
	total := decimal.Zero
	frozen := make(map[string]decimal.Decimal, len(state.GetDelegateAllocations()))
	for _, alloc := range state.GetDelegateAllocations() {
		amount := bytesToDec(alloc.GetVoterAmountFrozen())
		total = total.Add(amount)
		id, err := addressString(alloc.GetCandidateIdentifier())
		if err != nil {
			return errors.Wrap(err, "decode allocation candidate identifier")
		}
		frozen[id] = amount
	}
	row.TotalFrozen = total

	_, optedIn, err := p.snapshotDelegateConfigs(ctx, client, era, height, frozen)
	if err != nil {
		return err
	}
	row.DelegateCount = uint32(optedIn)
	p.pendingEraSeen[era] = true
	return nil
}

// isEraSeen and isEraDone answer "have we already done this work", counting
// both what is committed and what the current uncommitted batch has produced.
// The split matters only when a commit fails: commit() then drops the pending
// half, so the replayed batch redoes the work instead of skipping it.
func (p *Plugin) isEraSeen(era uint64) bool { return p.eraSeen[era] || p.pendingEraSeen[era] }
func (p *Plugin) isEraDone(era uint64) bool { return p.eraDone[era] || p.pendingEraDone[era] }
func (p *Plugin) isEraStatePassed(era uint64) bool {
	return p.eraStatePass[era] || p.pendingEraStatePass[era]
}

// snapshotDelegateConfigs writes one delegate_reward_config row per candidate
// and returns the freeze height read off the snapshots (0 when no candidate had
// one) together with how many of those candidates were opted in.
//
// The opted-in count is what stamps voter_reward_era.delegate_count. It is the
// size of the era's settlement set, which is deliberately not the number of
// delegates that ended up paying voters: a delegate frozen at 100% commission
// produces no distribution row at all, so counting payouts would erase it. The
// payout count stays derivable from voter_reward_distribution.delegate_id; this
// one is not derivable from anywhere else.
func (p *Plugin) snapshotDelegateConfigs(
	ctx context.Context, client iotexapi.APIServiceClient, era, height uint64,
	frozen map[string]decimal.Decimal,
) (uint64, int, error) {
	var (
		freezeHeight uint64
		optedIn      int
	)
	for _, info := range p.candidates.All() {
		routing, err := ReadPayoutAddress(ctx, client, info.Identifier, height)
		switch {
		case err == nil:
		case errors.Is(err, ErrNoPayoutRouting):
			// A candidate that vanished between the index refresh and this
			// read is not a reason to fail the block.
			log.L().Debug("voter_reward: delegate has no payout routing",
				zap.String("delegate", info.Identifier), zap.Error(err))
			continue
		default:
			// A failed read is not "this delegate has no routing". Skipping on
			// it would leave the delegate with no config row for this era, and
			// the era memo means nothing would ever go back for it.
			return 0, 0, errors.Wrapf(err, "read payout routing for %s at era %d", info.Identifier, era)
		}
		cfg := models.DelegateRewardConfig{
			DelegateID:    info.Identifier,
			Era:           era,
			BlockHeight:   height,
			DelegateName:  info.Name,
			OptedIn:       routing.OnchainRewardEnabled,
			PayoutAddress: routing.Address,
			TotalWeight:   decimal.Zero,
		}
		if amount, ok := frozen[info.Identifier]; ok {
			cfg.VoterAmountFrozen = amount
		} else {
			cfg.VoterAmountFrozen = decimal.Zero
		}
		if routing.OnchainRewardEnabled {
			optedIn++
			if h, ok := p.optInSeen[info.Identifier]; ok {
				cfg.OptInSource, cfg.OptInHeight = optInSourceAction, h
			} else {
				// Opted in with no SetVoterRewardOptIn ever observed. This
				// plugin always indexes from genesis (it deliberately does not
				// implement CatchUpSafe), so "never observed" really does mean
				// the fork-block Hermes migration, which emits no action and
				// no log. Height 0 records that there is no action to point
				// at, rather than attributing it to whichever block we
				// happened to notice in.
				cfg.OptInSource, cfg.OptInHeight = optInSourceMigration, 0
			}
		}
		snap, err := ReadDelegateSnapshot(ctx, client, info.Identifier, height)
		switch {
		case err == nil:
			freezeHeight = snap.FreezeHeight
			cfg.FreezeHeight = snap.FreezeHeight
			cfg.BlockCommissionBps = snap.BlockCommissionBps
			cfg.EpochCommissionBps = snap.EpochCommissionBps
			cfg.CommissionConfigured = snap.CommissionConfigured
			cfg.TotalWeight = dec(snap.TotalWeight)
			if !snap.CommissionConfigured {
				// Opted in but never configured its portions in the
				// DelegateProfile contract, so the protocol froze it at 100%
				// commission and its voters get nothing this era. Silent zero
				// rows downstream are the confusing symptom; say it here.
				log.L().Warn("voter_reward: delegate opted in without commission portions; voters receive nothing",
					zap.String("delegate", info.Identifier),
					zap.String("name", info.Name),
					zap.Uint64("era", era))
			}
		case errors.Is(err, ErrNoSnapshot):
			// Not opted in at the freeze; the zero values are correct.
		default:
			return 0, 0, err
		}
		p.pendingConfigs = append(p.pendingConfigs, cfg)
	}
	return freezeHeight, optedIn, nil
}

// indexFrozenEra records an era's frozen configuration from chain state at the
// era-boundary epoch's last block, for the eras the cursor path can never see.
//
// The protocol writes a distribution cursor only when at least one delegate has
// a non-zero voter allocation. An era in which every opted-in delegate sits at
// 100% commission produces no cursor at all — and that is exactly where a
// Hermes-migrated delegate lands until it publishes its portions in
// DelegateProfile, because the fork-block migration opts delegates in without
// checking whether they have any. ensureEraPlan gates on
// `state.GetTargetEra() == era`, so for such an era it can never fire: the
// freeze happened, every snapshot is readable on chain, and yet
// voter_reward_era and delegate_reward_config stay empty forever.
//
// Observed on TestNet at the first Zanzibar era (freeze 46,892,881, settlement
// 46,895,040): four delegates opted in, three of them frozen at 100%
// commission and the fourth outside the reward set, so the total voter
// allocation was zero, no cursor was written, and all four tables held zero
// rows while the chain reported optedIn=true for all four.
//
// The era number is the boundary epoch itself — the protocol stamps the cursor
// with `TargetEra: epochNum` — so it can be derived here without a cursor to
// read it from.
func (p *Plugin) indexFrozenEra(ctx context.Context, height, epoch uint64) error {
	if !protocol.IsEraBoundary(epoch, config.Default.Genesis.EpochsPerRewardEra) {
		return nil
	}
	// One pass per era, on the block that closes it: by then the freeze has
	// long happened and every snapshot for the era is readable.
	//
	// isEraStatePassed, not isEraSeen, is what makes this idempotent. This pass
	// runs on the boundary block and the first chunk lands on the very next one,
	// so claiming the eraSeen memo here would make ensureEraPlan skip every era
	// that does have a cursor — and ensureEraPlan owns the half of the row this
	// pass cannot know: first_chunk_at, total_frozen, and the per-delegate
	// voter_amount_frozen. isEraSeen is still consulted, but only to stand down
	// when the cursor path already recorded the plan: it reads strictly more
	// than this one does, so there would be nothing to add.
	if height != kernel.GetEpochLastBlockHeight(epoch) ||
		p.isEraStatePassed(epoch) || p.isEraSeen(epoch) {
		return nil
	}
	client := kernel.ChainClient()
	if client == nil {
		return errors.New("voter_reward: no chain client configured")
	}
	freezeHeight, optedIn, err := p.snapshotDelegateConfigs(ctx, client, epoch, height, nil)
	if err != nil {
		return err
	}
	if freezeHeight == 0 {
		// No candidate carried a snapshot, so nothing was frozen for this era
		// and there is no era to record. Leave the memo unset: a later block
		// is not going to change this, but neither is an empty row useful.
		return nil
	}
	row := p.era(epoch)
	row.FreezeHeight = freezeHeight
	// The opted-in count, not len(p.pendingConfigs): that slice is the whole
	// uncommitted batch, so it counts every config row buffered since the last
	// flush — other eras included — and grows with the batch size rather than
	// describing this era.
	row.DelegateCount = uint32(optedIn)
	if row.TotalFrozen.IsZero() {
		row.TotalFrozen = decimal.Zero
	}
	p.pendingEraStatePass[epoch] = true
	log.L().Info("voter_reward: indexed era from state (no distribution cursor)",
		zap.Uint64("era", epoch),
		zap.Uint64("freezeHeight", freezeHeight),
		zap.Uint64("height", height))
	return nil
}

func (p *Plugin) refreshCandidatesIfNeeded(ctx context.Context, height, epoch uint64) error {
	if height != kernel.GetEpochHeight(epoch) && p.candidates.Height() != 0 {
		return nil
	}
	client := kernel.ChainClient()
	if client == nil {
		return errors.New("voter_reward: no chain client configured")
	}
	// Read one block back: the candidate set for this epoch is settled by the
	// previous block, and reading at the block being indexed would race the
	// node's own commit.
	at := height
	if at > 0 {
		at--
	}
	if err := p.candidates.Refresh(ctx, client, at); err != nil {
		return errors.Wrapf(err, "refresh candidate index at %d", at)
	}
	return nil
}

func (p *Plugin) commit() error {
	rows, dests, configs := p.pendingRows, p.pendingDestinations, dedupeConfigs(p.pendingConfigs)
	eras := make([]models.VoterRewardEra, 0, len(p.pendingEras))
	for _, era := range p.pendingEras {
		eras = append(eras, *era)
	}
	p.pendingRows, p.pendingDestinations, p.pendingConfigs = nil, nil, nil
	p.pendingEras = map[uint64]*models.VoterRewardEra{}
	tipHeight := p.tipHeight

	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if len(rows) > 0 {
			// DoNothing on the (action_hash, log_index, row_index) unique index:
			// a row is fully identified by where it sat in the receipt, so a
			// replay can only ever produce the identical row.
			if err := tx.Table(p.table(models.VoterRewardDistribution{})).
				Clauses(clause.OnConflict{
					Columns: []clause.Column{
						{Name: "action_hash"}, {Name: "log_index"}, {Name: "row_index"},
					},
					DoNothing: true,
				}).CreateInBatches(rows, 200).Error; err != nil {
				return err
			}
		}
		if len(dests) > 0 {
			if err := tx.Table(p.table(models.VoterRewardDestination{})).
				CreateInBatches(dests, 200).Error; err != nil {
				return err
			}
		}
		for _, era := range eras {
			// The era row is written once and then advanced by every chunk, so
			// the running totals accumulate across batches rather than
			// overwriting. Columns that describe the immutable plan are only
			// written when this batch actually read them.
			if err := upsertEra(tx, p.table(models.VoterRewardEra{}), era); err != nil {
				return err
			}
		}
		if len(configs) > 0 {
			// Update every column except the opt-in attribution, which is
			// monotonic: once a delegate has been seen opting in by an
			// explicit action, a later reconciliation that has forgotten the
			// action must not relabel it as a fork migration. Letting
			// UpdateAll overwrite these two silently rewrote history on every
			// restart.
			assignable := []string{
				"block_height", "delegate_name", "opted_in", "freeze_height",
				"block_commission_bps", "epoch_commission_bps",
				"commission_configured", "total_weight", "voter_amount_frozen",
				"payout_address",
			}
			configTable := p.table(models.DelegateRewardConfig{})
			if err := tx.Table(configTable).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "delegate_id"}, {Name: "era"}},
				DoUpdates: clause.Assignments(assignmentsFor(assignable)),
			}).CreateInBatches(configs, 200).Error; err != nil {
				return err
			}
			if err := upgradeOptInAttribution(tx, configTable, configs); err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, p.Name(), tipHeight)
	})
	if err != nil {
		p.dropPendingMemos()
		return err
	}
	p.promotePendingMemos()
	return nil
}

// promotePendingMemos makes the current batch's memoisation durable, once its
// rows are actually in the database.
func (p *Plugin) promotePendingMemos() {
	for era := range p.pendingEraSeen {
		p.eraSeen[era] = true
	}
	for era := range p.pendingEraDone {
		p.eraDone[era] = true
	}
	for era := range p.pendingEraStatePass {
		p.eraStatePass[era] = true
	}
	clear(p.pendingEraSeen)
	clear(p.pendingEraDone)
	clear(p.pendingEraStatePass)
}

// dropPendingMemos discards the memoisation earned by a batch that failed to
// commit.
//
// The runner replays that exact batch. A memo kept here would describe work
// that is not in the database, and the replay would skip it — leaving the era
// row at freeze_height=0 and its delegate_reward_config rows absent, with
// nothing to ever go back for them.
func (p *Plugin) dropPendingMemos() {
	clear(p.pendingEraSeen)
	clear(p.pendingEraDone)
	clear(p.pendingEraStatePass)
}

// assignmentsFor builds the ON CONFLICT assignment map for the given columns,
// taking each value from the row that would have been inserted.
func assignmentsFor(columns []string) map[string]interface{} {
	out := make(map[string]interface{}, len(columns))
	for _, c := range columns {
		out[c] = gorm.Expr("excluded." + c)
	}
	return out
}

// upgradeOptInAttribution writes the opt-in source only where it strengthens
// what is already stored: an empty attribution can become anything, and a
// migration can be upgraded to an action, but never the reverse.
func upgradeOptInAttribution(tx *gorm.DB, table string, configs []models.DelegateRewardConfig) error {
	for _, cfg := range configs {
		if cfg.OptInSource == "" {
			continue
		}
		q := tx.Table(table).
			Where("delegate_id = ? AND era = ?", cfg.DelegateID, cfg.Era)
		if cfg.OptInSource == optInSourceMigration {
			// Do not clobber a recorded action with a migration.
			q = q.Where("opt_in_source = ?", "")
		} else {
			q = q.Where("opt_in_source <> ?", optInSourceAction)
		}
		if err := q.Updates(map[string]interface{}{
			"opt_in_source": cfg.OptInSource,
			"opt_in_height": cfg.OptInHeight,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func upsertEra(tx *gorm.DB, table string, era models.VoterRewardEra) error {
	var existing models.VoterRewardEra
	err := tx.Table(table).Where("era = ?", era.Era).Take(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return tx.Table(table).Create(&era).Error
	case err != nil:
		return err
	}
	existing.VoterCount += era.VoterCount
	existing.TotalDistributed = existing.TotalDistributed.Add(era.TotalDistributed)
	if era.LastChunkAt > existing.LastChunkAt {
		existing.LastChunkAt = era.LastChunkAt
		existing.ScanPhase = era.ScanPhase
		existing.ResumeVoter = era.ResumeVoter
	}
	if era.FreezeHeight != 0 {
		existing.FreezeHeight = era.FreezeHeight
		existing.DelegateCount = era.DelegateCount
		existing.TotalFrozen = era.TotalFrozen
	}
	if era.FirstChunkAt != 0 && existing.FirstChunkAt == 0 {
		existing.FirstChunkAt = era.FirstChunkAt
	}
	if era.CompletedHeight != 0 {
		existing.CompletedHeight = era.CompletedHeight
	}
	if !era.OverrunResidue.IsZero() {
		existing.OverrunResidue = era.OverrunResidue
	}
	return tx.Table(table).Save(&existing).Error
}

// dedupeConfigs collapses repeated (delegate_id, era) rows, keeping the last.
//
// Both passes that write an era's configs can land in the same batch: the
// boundary pass runs on the block that closes the era and the first chunk on the
// very next one, so a 200-block batch routinely spans both. Postgres rejects an
// INSERT .. ON CONFLICT DO UPDATE whose payload names the same conflict target
// twice -- "ON CONFLICT DO UPDATE command cannot affect row a second time" --
// and the runner then replays that batch forever.
//
// Last wins because the passes append in block order and the later one is the
// cursor path, which is the only one that knows each delegate's frozen voter
// amount.
func dedupeConfigs(configs []models.DelegateRewardConfig) []models.DelegateRewardConfig {
	if len(configs) < 2 {
		return configs
	}
	type key struct {
		delegate string
		era      uint64
	}
	at := make(map[key]int, len(configs))
	out := make([]models.DelegateRewardConfig, 0, len(configs))
	for _, c := range configs {
		k := key{c.DelegateID, c.Era}
		if i, seen := at[k]; seen {
			out[i] = c
			continue
		}
		at[k] = len(out)
		out = append(out, c)
	}
	return out
}
