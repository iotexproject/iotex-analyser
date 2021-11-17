package main

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.1.1"

const (
	// h := hash.Hash160b([]byte("staking"))
	// stakingProtocolAddr, err := address.FromBytes(h[:])
	// if err != nil {
	// 	return err
	// }
	StakingProtocolAddress = "io1qnpz47hx5q6r3w876axtrn6yz95d70cjl35r53"
)

type stakingActionPlugin struct {
}

func (b stakingActionPlugin) Name() string {
	return "staking_actions"
}

func (b stakingActionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b stakingActionPlugin) DependentPlugin() string {
	return "candidate"
}

func (b stakingActionPlugin) Start(ctx context.Context) error {
	if err := db.DB().AutoMigrate(&models.StakingActions{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b stakingActionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {

	err := db.DB().Transaction(func(tx *gorm.DB) error {
		var stakingAction models.StakingActions
		actions := make(map[hash.Hash256]action.SealedEnvelope, len(blk.Actions))
		bucketMap := make(map[string]uint64)
		for _, selp := range blk.Actions {
			actionHash := selp.Hash()
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
			act := selp.Action()
			actionHash := selp.Hash()
			actHash := hex.EncodeToString(actionHash[:])
			cmpNum := big.NewInt(100000000)
			for _, log := range receipt.Logs() {
				if log.Address == StakingProtocolAddress && len(log.Topics) > 1 {
					bucketIndex := new(big.Int).SetBytes(log.Topics[1][:])

					if bucketIndex.Cmp(cmpNum) > 0 {
						continue
					}
					bucketMap[actHash] = bucketIndex.Uint64()
				}
			}
			switch a := act.(type) {
			case *action.CreateStake:
				cadidateAddr, err := getCandidateAddressByName(a.Candidate(), blk.Height())
				if err != nil {
					return err
				}

				bucketID, ok := bucketMap[actHash]
				if !ok {
					return errors.New("can not found bucketID with actHash:" + actHash)
				}
				stakingAction = models.StakingActions{
					BlockHeight:  blk.Height(),
					BucketID:     bucketID,
					Sender:       sender.String(),
					OwnerAddress: sender.String(),
					ActHash:      actHash,
					Candidate:    cadidateAddr,
					Amount:       decimal.NewFromBigInt(a.Amount(), 0),
					ActType:      "StakeCreate",
					AutoStake:    a.AutoStake(),
					Duration:     a.Duration(),
				}
				if err := tx.Create(&stakingAction).Error; err != nil {
					return err
				}
			case *action.TransferStake:
				bucketID := a.BucketIndex()
				decmailAmount, err := getBucketSumAmountByBucketID(tx, bucketID)
				if err != nil {
					return errors.Wrapf(err, "getBucketSumAmountByBucketID error, bucketID: %d", bucketID)
				}
				info, err := getBucketInfoAddressByBucketID(tx, bucketID)
				if err != nil {
					return errors.Wrap(err, "getBucketInfoAddressByBucketID error")
				}
				stakingAction = models.StakingActions{
					BlockHeight:  blk.Height(),
					BucketID:     bucketID,
					OwnerAddress: info.OwnerAddress,
					Sender:       sender.String(),
					ActHash:      actHash,
					Candidate:    info.Candidate,
					AutoStake:    info.AutoStake,
					ActType:      "TransferStake",
					Duration:     info.Duration,
					Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
				}
				if err := tx.Create(&stakingAction).Error; err != nil {
					return err
				}
				stakingAction = models.StakingActions{
					BlockHeight:  blk.Height(),
					BucketID:     bucketID,
					OwnerAddress: a.VoterAddress().String(),
					Sender:       sender.String(),
					ActHash:      actHash,
					Candidate:    info.Candidate,
					AutoStake:    info.AutoStake,
					ActType:      "TransferStake",
					Duration:     info.Duration,
					Amount:       decmailAmount,
				}
				if err := tx.Create(&stakingAction).Error; err != nil {
					return err
				}
			case *action.Restake:
				bucketID := a.BucketIndex()
				info, err := getBucketInfoAddressByBucketID(tx, bucketID)
				if err != nil {
					return errors.Wrap(err, "getBucketInfoAddressByBucketID error")
				}
				// fix greenland (height=6544441) restake
				fixAmount := decimal.NewFromInt(0)
				if blk.Height() < genesis.Default.GreenlandBlockHeight {
					fixAmount, err = getFixBucketSumAmountByBucketID(tx, bucketID)
					if err != nil {
						return errors.Wrapf(err, "getBucketSumAmountByBucketID error, bucketID: %d", bucketID)
					}
				}
				stakingAction = models.StakingActions{
					BlockHeight:  blk.Height(),
					BucketID:     bucketID,
					OwnerAddress: sender.String(),
					Sender:       sender.String(),
					ActHash:      actHash,
					Candidate:    info.Candidate,
					AutoStake:    a.AutoStake(),
					ActType:      "Restake",
					Duration:     a.Duration(),
					Amount:       fixAmount,
				}
				if err := tx.Create(&stakingAction).Error; err != nil {
					return err
				}
			case *action.ChangeCandidate:
				bucketID := a.BucketIndex()
				decmailAmount, err := getBucketSumAmountByBucketID(tx, bucketID)
				if err != nil {
					return errors.Wrapf(err, "getBucketSumAmountByBucketID error, bucketID: %d", bucketID)
				}
				info, err := getBucketInfoAddressByBucketID(tx, bucketID)
				if err != nil {
					return errors.Wrap(err, "getBucketInfoAddressByBucketID error")
				}
				stakingAction = models.StakingActions{
					BlockHeight:  blk.Height(),
					BucketID:     bucketID,
					OwnerAddress: info.OwnerAddress,
					Sender:       sender.String(),
					ActHash:      actHash,
					Candidate:    info.Candidate,
					AutoStake:    info.AutoStake,
					ActType:      "ChangeCandidate",
					Duration:     info.Duration,
					Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
				}
				if err := tx.Create(&stakingAction).Error; err != nil {
					return err
				}
				cadidateAddr, err := getCandidateAddressByName(a.Candidate(), blk.Height())
				if err != nil {
					return err
				}
				stakingAction = models.StakingActions{
					BlockHeight:  blk.Height(),
					BucketID:     bucketID,
					OwnerAddress: info.OwnerAddress,
					Sender:       sender.String(),
					ActHash:      actHash,
					Candidate:    cadidateAddr,
					AutoStake:    info.AutoStake,
					ActType:      "ChangeCandidate",
					Duration:     info.Duration,
					Amount:       decmailAmount,
				}
				if err := tx.Create(&stakingAction).Error; err != nil {
					return err
				}
			case *action.DepositToStake:
				bucketID := a.BucketIndex()
				info, err := getBucketInfoAddressByBucketID(tx, bucketID)
				if err != nil {
					return errors.Wrap(err, "getBucketInfoAddressByBucketID error")
				}
				stakingAction = models.StakingActions{
					BlockHeight:  blk.Height(),
					BucketID:     bucketID,
					OwnerAddress: info.OwnerAddress,
					Sender:       sender.String(),
					ActHash:      actHash,
					Candidate:    info.Candidate,
					AutoStake:    info.AutoStake,
					ActType:      "DepositToStake",
					Duration:     info.Duration,
					Amount:       decimal.NewFromBigInt(a.Amount(), 0),
				}
				if err := tx.Create(&stakingAction).Error; err != nil {
					return err
				}
			case *action.Unstake:
				bucketID := a.BucketIndex()
				decmailAmount, err := getBucketSumAmountByBucketID(tx, bucketID)
				if err != nil {
					return errors.Wrap(err, "getBucketSumAmountByBucketID error")
				}
				info, err := getBucketInfoAddressByBucketID(tx, bucketID)
				if err != nil {
					return errors.Wrap(err, "getBucketInfoAddressByBucketID error")
				}
				stakingAction = models.StakingActions{
					BlockHeight:  blk.Height(),
					BucketID:     bucketID,
					OwnerAddress: sender.String(),
					Sender:       sender.String(),
					ActHash:      actHash,
					Candidate:    info.Candidate,
					AutoStake:    info.AutoStake,
					ActType:      "Unstake",
					Duration:     info.Duration,
					Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
				}
				if err := tx.Create(&stakingAction).Error; err != nil {
					return err
				}
			case *action.CandidateRegister:
				bucketID, ok := bucketMap[actHash]
				if !ok {
					return errors.New("can not found bucketID with actHash:" + actHash)
				}
				stakingAction = models.StakingActions{
					BlockHeight:  blk.Height(),
					BucketID:     bucketID,
					Sender:       sender.String(),
					OwnerAddress: a.OwnerAddress().String(),
					ActHash:      actHash,
					Candidate:    a.OwnerAddress().String(),
					Amount:       decimal.NewFromBigInt(a.Amount(), 0),
					ActType:      "CandidateRegister",
					AutoStake:    a.AutoStake(),
					Duration:     a.Duration(),
				}
				if err := tx.Create(&stakingAction).Error; err != nil {
					return err
				}
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return errors.Wrap(err, "")
}

func (b stakingActionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b stakingActionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = stakingActionPlugin{}
