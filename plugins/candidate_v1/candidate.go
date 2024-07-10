package main

import (
	"context"
	"math/big"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	slog "github.com/iotexproject/iotex-core/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
)

const VERSION = "2.0.7"

var CandidateEndorsementOpRevokeHash256 hash.Hash256

type candidatePlugin struct {
	batchSize             int
	tipHeight             uint64
	candidateDatas []*Candidate
}

func (b candidatePlugin) Name() string {
	return "candidate_v1"
}

func (b candidatePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b candidatePlugin) BatchSize() int {
	return b.batchSize
}


func (b *candidatePlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &Candidate{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	CandidateEndorsementOpRevokeHash256 = hash.BytesToHash256([]byte("candidateEndorsementWithOp"))

	return nil
}

func (b *candidatePlugin) handleAction(act action.Action, logs []*action.Log, blkHeight uint64, sender address.Address) error {
	switch a := act.(type) {
	case *action.CandidateRegister:
		createData := &Candidate{
			BlockHeight:     blkHeight,
			Name:            a.Name(),
			OperatorAddress: a.OperatorAddress().String(),
			OwnerAddress:    a.OwnerAddress().String(),
			CandidateID:     a.OwnerAddress().String(),
			RewardAddress:   a.RewardAddress().String(),
			Amount:          decimal.NewFromBigInt(a.Amount(), 0),
			ActType:         "CandidateRegister",
			AutoStake:       a.AutoStake(),
			Duration:        a.Duration(),
			Payload:         a.Payload(),
		}
		b.candidateDatas = append(b.candidateDatas, createData)
	case *action.CandidateUpdate:
		rewardAddress := ""
		if a.RewardAddress() != nil {
			rewardAddress = a.RewardAddress().String()
		}

		candidate := &Candidate{}
		if err := candidate.FetchByNameWithHeight(a.Name(), blkHeight, b.candidateDatas); err != nil {
			return err
		}

		createData := &Candidate{
			BlockHeight:     blkHeight,
			Name:            a.Name(),
			OperatorAddress: a.OperatorAddress().String(),
			OwnerAddress:    sender.String(),
			RewardAddress:   rewardAddress,
			CandidateID:     candidate.CandidateID,
			ActType:         "CandidateUpdate",
		}
		b.candidateDatas = append(b.candidateDatas, createData)
	case *action.CandidateTransferOwnership:
		newOwner := ""
		if a.NewOwner() != nil {
			newOwner = a.NewOwner().String()
		}

		candidate := &Candidate{}
		if err := candidate.FetchByOwnerAddressWithHeight(sender.String(), blkHeight, b.candidateDatas); err != nil {
			return err
		}

		createData := &Candidate{
			BlockHeight:     blkHeight,
			Name:            candidate.Name,
			OperatorAddress: candidate.OperatorAddress,
			RewardAddress:   candidate.RewardAddress,
			OwnerAddress:    newOwner,
			CandidateID:     candidate.CandidateID,
			ActType:         "CandidateTransferOwnership",
		}
		b.candidateDatas = append(b.candidateDatas, createData)
	// navtive bucket
	case *action.CandidateActivate:
		bucketID := a.BucketID()
		bucket, err := GetStakingBucketByID(bucketID, blkHeight)
		if err != nil {
			return err
		}

		candidate := &Candidate{}
		if err := candidate.FetchByCandidateIDWithHeight(bucket.CandidateAddress, blkHeight, b.candidateDatas); err != nil {
			return err
		}

		// TODO amount is the staked amount at the epoch height not block height
		amount, ok := new(big.Int).SetString(bucket.StakedAmount, 10)
		if !ok {
			return errors.New("failed to parse bucket staked amount")
		}
		createData := &Candidate{
			BlockHeight:     blkHeight,
			Name:            candidate.Name,
			OperatorAddress: candidate.OperatorAddress,
			OwnerAddress:    candidate.OwnerAddress,
			CandidateID:     candidate.CandidateID,
			RewardAddress:   candidate.RewardAddress,
			Amount:          decimal.NewFromBigInt(amount, 0),
			ActType:         "CandidateActivate",
			AutoStake:       bucket.AutoStake,
			Duration:        bucket.StakedDuration,
			Payload:         candidate.Payload,
		}
		b.candidateDatas = append(b.candidateDatas, createData)
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
					candidate := &Candidate{}
					if err := candidate.FetchByCandidateIDWithHeight(candidateID.String(), blkHeight, b.candidateDatas); err != nil {
						return err
					}
					createData := &Candidate{
						BlockHeight:     blkHeight,
						Name:            candidate.Name,
						OperatorAddress: candidate.OperatorAddress,
						RewardAddress:   candidate.RewardAddress,
						OwnerAddress:    candidate.OwnerAddress,
						CandidateID:     candidate.CandidateID,
						ActType:         "CandidateEndorsement-CandidateEndorsementOpRevoke",
					}
					b.candidateDatas = append(b.candidateDatas, createData)
				}
			}
		}
	}
	return nil
}

func (b *candidatePlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}

	b.tipHeight = blks[0].Height() + uint64(len(blks)) - 1
	return b.commit()
}

func (b candidatePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *candidatePlugin) putBlock(ctx context.Context, blk *block.Block) error {
	actions := make(map[hash.Hash256]*action.SealedEnvelope, len(blk.Actions))
		for _, selp := range blk.Actions {
			actionHash, _ := selp.Hash()
			actions[actionHash] = selp
		}

		for _, receipt := range blk.Receipts {
			if receipt.Status != uint64(iotextypes.ReceiptStatus_Success) {
				continue
			}
			selp, ok := actions[receipt.ActionHash]
			if !ok {
				continue
			}
			sender, _ := address.FromBytes(selp.SrcPubkey().Hash())
			b.handleAction(selp.Action(), receipt.Logs(), blk.Height(), sender)
		}

	return nil
}

func (b *candidatePlugin) commit() error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if len(b.candidateDatas) > 0 {
			if err := db.DB().Model(&Candidate{}).CreateInBatches(b.candidateDatas, 2000).Error; err != nil {
				slog.L().Error("put candidateDatas ", zap.String("plugin", b.Name()), zap.Int("size", len(b.candidateDatas)))
				b.candidateDatas = b.candidateDatas[:0]
				return err
			}
			b.candidateDatas = b.candidateDatas[:0]
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), b.tipHeight)
	})
	return err
}

func (b candidatePlugin) Stop(ctx context.Context) error {
	return nil
}

func (b candidatePlugin) Version() string {
	return VERSION
}

// exported
var Plugin = candidatePlugin{}
