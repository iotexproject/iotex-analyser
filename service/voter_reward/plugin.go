package voter_reward

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
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
}

// New returns a plugin instance ready to be registered.
//
// It returns a value, not a pointer, because the loader resolves the exported
// `Plugin` symbol with plugin.Lookup, which hands back the *address of the
// package-level variable*. Exporting a `*Plugin` would therefore make the
// symbol a `**Plugin` and fail the Adapter type assertion.
func New(shadow plugin.PluginShadow) Plugin {
	return Plugin{
		Shadow:      shadow,
		candidates:  NewCandidateIndex(),
		optInSeen:   map[string]uint64{},
		eraSeen:     map[uint64]bool{},
		eraDone:     map[uint64]bool{},
		chunkEras:   map[uint64]struct{}{},
		pendingEras: map[uint64]*models.VoterRewardEra{},
	}
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
	return p.refreshChunkedEras(ctx, height)
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
		if !p.eraDone[era] {
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
	resume, err := hexOrEmpty(state.GetResumeVoter())
	if err != nil {
		return err
	}
	row.ResumeVoter = resume
	if state.GetCompleted() || state.GetScanPhase() == ScanPhaseDone {
		if !p.eraDone[era] {
			p.eraDone[era] = true
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
	if p.eraSeen[era] {
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
	row.DelegateCount = uint32(len(state.GetDelegateAllocations()))
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

	if err := p.snapshotDelegateConfigs(ctx, client, era, height, frozen); err != nil {
		return err
	}
	p.eraSeen[era] = true
	return nil
}

func (p *Plugin) snapshotDelegateConfigs(
	ctx context.Context, client iotexapi.APIServiceClient, era, height uint64,
	frozen map[string]decimal.Decimal,
) error {
	for _, info := range p.candidates.All() {
		routing, err := ReadPayoutAddress(ctx, client, info.Identifier, height)
		if err != nil {
			// A candidate that vanished between the index refresh and this
			// read is not a reason to fail the block.
			log.L().Debug("voter_reward: payout address unreadable",
				zap.String("delegate", info.Identifier), zap.Error(err))
			continue
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
			return err
		}
		p.pendingConfigs = append(p.pendingConfigs, cfg)
	}
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
	rows, dests, configs := p.pendingRows, p.pendingDestinations, p.pendingConfigs
	eras := make([]models.VoterRewardEra, 0, len(p.pendingEras))
	for _, era := range p.pendingEras {
		eras = append(eras, *era)
	}
	p.pendingRows, p.pendingDestinations, p.pendingConfigs = nil, nil, nil
	p.pendingEras = map[uint64]*models.VoterRewardEra{}
	tipHeight := p.tipHeight

	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(rows) > 0 {
			if err := tx.CreateInBatches(rows, 200).Error; err != nil {
				return err
			}
		}
		if len(dests) > 0 {
			if err := tx.CreateInBatches(dests, 200).Error; err != nil {
				return err
			}
		}
		for _, era := range eras {
			// The era row is written once and then advanced by every chunk, so
			// the running totals accumulate across batches rather than
			// overwriting. Columns that describe the immutable plan are only
			// written when this batch actually read them.
			if err := upsertEra(tx, era); err != nil {
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
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "delegate_id"}, {Name: "era"}},
				DoUpdates: clause.Assignments(assignmentsFor(assignable)),
			}).CreateInBatches(configs, 200).Error; err != nil {
				return err
			}
			if err := upgradeOptInAttribution(tx, configs); err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, p.Name(), tipHeight)
	})
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
func upgradeOptInAttribution(tx *gorm.DB, configs []models.DelegateRewardConfig) error {
	for _, cfg := range configs {
		if cfg.OptInSource == "" {
			continue
		}
		q := tx.Model(&models.DelegateRewardConfig{}).
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

func upsertEra(tx *gorm.DB, era models.VoterRewardEra) error {
	var existing models.VoterRewardEra
	err := tx.Where("era = ?", era.Era).Take(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return tx.Create(&era).Error
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
	return tx.Save(&existing).Error
}
