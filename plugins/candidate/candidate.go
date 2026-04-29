package main

import (
	"context"
	"math/big"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
)

const VERSION = "2.0.7"

var CandidateEndorsementOpRevokeHash256 hash.Hash256

type candidatePlugin struct {
}

func (b candidatePlugin) Name() string {
	return "candidate"
}

func (b candidatePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b candidatePlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.Candidate{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	CandidateEndorsementOpRevokeHash256 = hash.BytesToHash256([]byte("candidateEndorsementWithOp"))

	return nil
}

func handleAction(act action.Action, logs []*action.Log, blkHeight uint64, sender address.Address, tx *gorm.DB) error {
	switch a := act.(type) {
	case *action.CandidateRegister:
		createData := models.Candidate{
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
		if err := tx.Create(&createData).Error; err != nil {
			return err
		}
	case *action.CandidateUpdate:
		rewardAddress := ""
		if a.RewardAddress() != nil {
			rewardAddress = a.RewardAddress().String()
		}

		// CandidateUpdate is accepted by the chain when signed by either the
		// owner or the operator (e.g. testnet block 43063855 — operator was
		// changed at 43063850 and the new operator immediately calls
		// CandidateUpdate). Look up the candidate by either field.
		candidate := &models.Candidate{}
		if err := candidate.FetchByOwnerOrOperatorWithHeight(sender.String(), blkHeight, tx); err != nil {
			return err
		}

		createData := models.Candidate{
			BlockHeight:     blkHeight,
			Name:            a.Name(),
			OperatorAddress: a.OperatorAddress().String(),
			OwnerAddress:    candidate.OwnerAddress, // preserve existing owner; operator-signed updates must NOT rewrite the owner field
			RewardAddress:   rewardAddress,
			CandidateID:     candidate.CandidateID,
			ActType:         "CandidateUpdate",
		}
		if err := tx.Create(&createData).Error; err != nil {
			return err
		}
	case *action.CandidateTransferOwnership:
		newOwner := ""
		if a.NewOwner() != nil {
			newOwner = a.NewOwner().String()
		}

		// CandidateTransferOwnership is also accepted from owner or operator
		// per the staking protocol; same lookup pattern as CandidateUpdate.
		candidate := &models.Candidate{}
		if err := candidate.FetchByOwnerOrOperatorWithHeight(sender.String(), blkHeight, tx); err != nil {
			return err
		}

		createData := models.Candidate{
			BlockHeight:     blkHeight,
			Name:            candidate.Name,
			OperatorAddress: candidate.OperatorAddress,
			RewardAddress:   candidate.RewardAddress,
			OwnerAddress:    newOwner,
			CandidateID:     candidate.CandidateID,
			ActType:         "CandidateTransferOwnership",
		}
		if err := tx.Create(&createData).Error; err != nil {
			return err
		}
	// navtive bucket
	case *action.CandidateActivate:
		bucketID := a.BucketID()
		bucket, err := GetStakingBucketByID(bucketID, blkHeight)
		if err != nil {
			return err
		}

		candidate := &models.Candidate{}
		if err := candidate.FetchByCandidateIDWithHeight(bucket.CandidateAddress, blkHeight, tx); err != nil {
			return err
		}

		// TODO amount is the staked amount at the epoch height not block height
		amount, ok := new(big.Int).SetString(bucket.StakedAmount, 10)
		if !ok {
			return errors.New("failed to parse bucket staked amount")
		}
		createData := models.Candidate{
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
		if err := tx.Create(&createData).Error; err != nil {
			return err
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
					candidate := &models.Candidate{}
					if err := candidate.FetchByCandidateIDWithHeight(candidateID.String(), blkHeight, tx); err != nil {
						return err
					}
					createData := models.Candidate{
						BlockHeight:     blkHeight,
						Name:            candidate.Name,
						OperatorAddress: candidate.OperatorAddress,
						RewardAddress:   candidate.RewardAddress,
						OwnerAddress:    candidate.OwnerAddress,
						CandidateID:     candidate.CandidateID,
						ActType:         "CandidateEndorsement-CandidateEndorsementOpRevoke",
					}
					if err := tx.Create(&createData).Error; err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
func handleBlock(blk *block.Block, tx *gorm.DB) error {
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
		if err := handleAction(selp.Action(), receipt.Logs(), blk.Height(), sender, tx); err != nil {
			return err
		}
	}
	return nil
}

func (b candidatePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if err := handleBlock(blk, tx); err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
}

func (b candidatePlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	if len(blks) == 0 {
		return nil
	}
	return db.DB().Transaction(func(tx *gorm.DB) error {
		for _, blk := range blks {
			if err := handleBlock(blk, tx); err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blks[len(blks)-1].Height())
	})
}

func (b candidatePlugin) BatchSize() int {
	return 0 // use runner default (config.BlockDB.BatchSize)
}

func (b candidatePlugin) Stop(ctx context.Context) error {
	return nil
}

func (b candidatePlugin) Version() string {
	return VERSION
}

// exported
var Plugin = candidatePlugin{}
