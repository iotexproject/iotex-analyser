package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
)

const VERSION = "2.0.7"

var CandidateEndorsementOpRevokeHash256 hash.Hash256

type candidatePlugin struct {
	batchSize int
}

type stash struct {
	byID    map[string]*models.Candidate
	byOwner map[string]*models.Candidate
}

type Config struct {
	BatchSize int `yaml:"batchSize"`
}

func (b candidatePlugin) Name() string {
	return "candidate"
}

func (b candidatePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b candidatePlugin) BatchSize() int {
	return b.batchSize
}

func (b *candidatePlugin) Start(ctx context.Context) error {
	var err error
	cfg := &Config{
		BatchSize: 200,
	}
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		if err = yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}
	b.batchSize = cfg.BatchSize

	if err := db.ChConn().Exec(ctx, models.CandidateDDL); err != nil {
		return errors.Wrapf(err, "failed to create table %s", models.Candidate{}.TableName())
	}
	CandidateEndorsementOpRevokeHash256 = hash.BytesToHash256([]byte("candidateEndorsementWithOp"))
	return nil
}

func handleAction(act action.Action, logs []*action.Log, blkHeight uint64, sender address.Address, index uint32, stash *stash) (*models.Candidate, error) {
	var createData *models.Candidate
	switch a := act.(type) {
	case *action.CandidateRegister:
		createData = &models.Candidate{
			BlockHeight:     blkHeight,
			Name:            a.Name(),
			OperatorAddress: a.OperatorAddress().String(),
			OwnerAddress:    a.OwnerAddress().String(),
			CandidateID:     a.OwnerAddress().String(),
			RewardAddress:   a.RewardAddress().String(),
			Amount:          a.Amount().String(),
			ActType:         "CandidateRegister",
			AutoStake:       a.AutoStake(),
			Duration:        a.Duration(),
			Payload:         a.Payload(),
			LogIndex:        index,
		}
		stash.byID[a.OwnerAddress().String()] = createData
		stash.byOwner[a.OwnerAddress().String()] = createData
	case *action.CandidateUpdate:
		rewardAddress := ""
		if a.RewardAddress() != nil {
			rewardAddress = a.RewardAddress().String()
		}

		candidate, err := fetchCandidateByOwnerAt(stash.byOwner, sender.String(), blkHeight)
		if err != nil {
			return nil, err
		}

		createData = &models.Candidate{
			BlockHeight:     blkHeight,
			Name:            a.Name(),
			OperatorAddress: a.OperatorAddress().String(),
			OwnerAddress:    sender.String(),
			RewardAddress:   rewardAddress,
			CandidateID:     candidate.CandidateID,
			ActType:         "CandidateUpdate",
			LogIndex:        index,
		}
	case *action.CandidateTransferOwnership:
		newOwner := ""
		if a.NewOwner() != nil {
			newOwner = a.NewOwner().String()
		}

		candidate, err := fetchCandidateByOwnerAt(stash.byOwner, sender.String(), blkHeight)
		if err != nil {
			return nil, err
		}

		createData = &models.Candidate{
			BlockHeight:     blkHeight,
			Name:            candidate.Name,
			OperatorAddress: candidate.OperatorAddress,
			RewardAddress:   candidate.RewardAddress,
			OwnerAddress:    newOwner,
			CandidateID:     candidate.CandidateID,
			ActType:         "CandidateTransferOwnership",
			LogIndex:        index,
		}
		delete(stash.byOwner, sender.String())
		stash.byOwner[newOwner] = createData
	// navtive bucket
	case *action.CandidateActivate:
		bucketID := a.BucketID()
		bucket, err := GetStakingBucketByID(bucketID, blkHeight)
		if err != nil {
			return nil, err
		}

		candidate, err := fetchCandidateByIDAt(stash.byID, bucket.CandidateAddress, blkHeight)
		if err != nil {
			return nil, err
		}

		// TODO amount is the staked amount at the epoch height not block height
		amount, ok := new(big.Int).SetString(bucket.StakedAmount, 10)
		if !ok {
			return nil, errors.New("failed to parse bucket staked amount")
		}
		createData = &models.Candidate{
			BlockHeight:     blkHeight,
			Name:            candidate.Name,
			OperatorAddress: candidate.OperatorAddress,
			OwnerAddress:    candidate.OwnerAddress,
			CandidateID:     candidate.CandidateID,
			RewardAddress:   candidate.RewardAddress,
			Amount:          amount.String(),
			ActType:         "CandidateActivate",
			AutoStake:       bucket.AutoStake,
			Duration:        bucket.StakedDuration,
			Payload:         candidate.Payload,
			LogIndex:        index,
		}
	case *action.CandidateEndorsement:
		switch a.Op() {
		case action.CandidateEndorsementOpRevoke:
			var candidateID address.Address
			for _, log := range logs {
				if log.Address == "" || len(log.Topics) < 2 {
					continue
				}
				// candidateEndorsementWithOp(uint64, address, uint32)
				if log.Topics[0] == CandidateEndorsementOpRevokeHash256 {
					// candidateID = common.BytesToAddress(log.Topics[2][:])
					candidateID, _ = address.FromBytes(log.Topics[2][:])

					candidate, err := fetchCandidateByIDAt(stash.byID, candidateID.String(), blkHeight)
					if err != nil {
						return nil, err
					}
					createData = &models.Candidate{
						BlockHeight:     blkHeight,
						Name:            candidate.Name,
						OperatorAddress: candidate.OperatorAddress,
						RewardAddress:   candidate.RewardAddress,
						OwnerAddress:    candidate.OwnerAddress,
						CandidateID:     candidate.CandidateID,
						ActType:         "CandidateEndorsement-CandidateEndorsementOpRevoke",
						LogIndex:        index,
					}
				}
			}
		}
	}
	return createData, nil
}

func (b candidatePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	stash := &stash{
		byID:    make(map[string]*models.Candidate),
		byOwner: make(map[string]*models.Candidate),
	}
	cands, err := b.handleBlock(ctx, blk, stash)
	if err != nil {
		return errors.Wrapf(err, "failed to handle block %d", blk.Height())
	}
	return b.commit(ctx, cands, blk.Height())
}

func (b candidatePlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	total := make([]*models.Candidate, 0)
	stash := &stash{
		byID:    make(map[string]*models.Candidate),
		byOwner: make(map[string]*models.Candidate),
	}
	for _, blk := range blks {
		cands, err := b.handleBlock(ctx, blk, stash)
		if err != nil {
			return errors.Wrapf(err, "failed to handle block %d", blk.Height())
		}
		total = append(total, cands...)
	}
	return b.commit(ctx, total, blks[len(blks)-1].Height())
}

func (b candidatePlugin) handleBlock(ctx context.Context, blk *block.Block, stash *stash) ([]*models.Candidate, error) {
	actions := make(map[hash.Hash256]*action.SealedEnvelope, len(blk.Actions))
	for _, selp := range blk.Actions {
		actionHash, _ := selp.Hash()
		actions[actionHash] = selp
	}
	cands := make([]*models.Candidate, 0)
	for index, receipt := range blk.Receipts {
		if receipt.Status != uint64(iotextypes.ReceiptStatus_Success) {
			continue
		}
		selp, ok := actions[receipt.ActionHash]
		if !ok {
			continue
		}
		sender, _ := address.FromBytes(selp.SrcPubkey().Hash())
		m, err := handleAction(selp.Action(), receipt.Logs(), blk.Height(), sender, uint32(index), stash)
		if err != nil {
			return nil, err
		}
		if m != nil {
			cands = append(cands, m)
		}
	}
	return cands, nil
}

func (b candidatePlugin) commit(ctx context.Context, cands []*models.Candidate, height uint64) error {
	// batch insert to clickhouse
	batch, err := db.ChConn().PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", models.Candidate{}.TableName()))
	if err != nil {
		return errors.Wrap(err, "failed to prepare batch")
	}
	for _, e := range cands {
		if err := batch.AppendStruct(e); err != nil {
			return errors.Wrap(err, "failed to append struct")
		}
	}
	if err := batch.Send(); err != nil {
		return errors.Wrap(err, "failed to send batch")
	}
	// update index height
	return db.UpdateIndexHeight(b.Name(), height)

}

func (b candidatePlugin) Stop(ctx context.Context) error {
	return nil
}

func (b candidatePlugin) Version() string {
	return VERSION
}

// exported
var Plugin = candidatePlugin{}
