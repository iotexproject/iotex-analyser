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

const VERSION = "2.0.4"

const (
	// h := hash.Hash160b([]byte("staking"))
	// stakingProtocolAddr, err := address.FromBytes(h[:])
	// if err != nil {
	// 	return err
	// }
	StakingProtocolAddress  = "io1qnpz47hx5q6r3w876axtrn6yz95d70cjl35r53"
	GovernaceForwardAddress = "io1xfdn0z046hzm03jrtm8hf4scw2w07t7a0mqtmz"
	GovernaceForwardToHash  = "87f136fb149a2751c6cc2ee9d9d2f7fc786ac7a163f58ef9fdbe2f1d7e7c78e8"
	GovernaceCacelHash      = "dfae2e44eee3429afab9409ee9f946d11d84e8eee5d3c81525197a2925b0ceb9"
)

type stakingActionPlugin struct {
}

func (b stakingActionPlugin) Name() string {
	return "staking_action"
}

func (b stakingActionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b stakingActionPlugin) DependentPlugin() string {
	return "candidate"
}

func (b stakingActionPlugin) Start(ctx context.Context) error {
	if err := db.DB().AutoMigrate(&models.StakingAction{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b stakingActionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {

	err := db.DB().Transaction(func(tx *gorm.DB) error {
		var stakingAction models.StakingAction
		actions := make(map[hash.Hash256]action.SealedEnvelope, len(blk.Actions))
		bucketMap := make(map[string]uint64)
		for _, selp := range blk.Actions {
			actHash := selp.Hash()
			actions[actHash] = selp
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
				stakingAction = models.StakingAction{
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
				stakingAction = models.StakingAction{
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
				stakingAction = models.StakingAction{
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
				stakingAction = models.StakingAction{
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
				stakingAction = models.StakingAction{
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
				stakingAction = models.StakingAction{
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
				stakingAction = models.StakingAction{
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
				stakingAction = models.StakingAction{
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
				stakingAction = models.StakingAction{
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
			for _, log := range receipt.Logs() {
				if log.Address != GovernaceForwardAddress {
					continue
				}
				if len(log.Topics) < 2 {
					return errors.New("topics len must >= 2")
				}
				topics := log.Topics
				funcHash := hex.EncodeToString(topics[0][:])
				switch funcHash {
				case GovernaceForwardToHash:
					from, err := getAddresFromHash256(topics[1])
					if err != nil {
						return err
					}
					to, err := getAddresFromHash256(topics[2])
					if err != nil {
						return err
					}
					bucketIDs, err := getBucketIDsByAddressWithHeight(from.String(), blk.Height())
					if err != nil {
						return err
					}
					for _, bucketID := range bucketIDs {
						decmailAmount, err := getBucketSumAmountByBucketID(tx, bucketID)
						if err != nil {
							return errors.Wrapf(err, "getBucketSumAmountByBucketID error, bucketID: %d", bucketID)
						}
						info, err := getBucketInfoAddressByBucketID(tx, bucketID)
						if err != nil {
							return errors.Wrap(err, "getBucketInfoAddressByBucketID error")
						}
						if info.OwnerAddress != from.String() {
							continue
						}
						stakingAction = models.StakingAction{
							BlockHeight:  blk.Height(),
							BucketID:     bucketID,
							OwnerAddress: info.OwnerAddress,
							Sender:       sender.String(),
							ActHash:      actHash,
							ForwardTo:    to.String(),
							Candidate:    info.Candidate,
							AutoStake:    info.AutoStake,
							ActType:      "GovernaceForward",
							Duration:     info.Duration,
							Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
						}
						if err := tx.Create(&stakingAction).Error; err != nil {
							return err
						}
						stakingAction = models.StakingAction{
							BlockHeight:  blk.Height(),
							BucketID:     bucketID,
							OwnerAddress: to.String(),
							Sender:       sender.String(),
							ActHash:      actHash,
							Candidate:    info.Candidate,
							AutoStake:    info.AutoStake,
							ActType:      "GovernaceForward",
							Duration:     info.Duration,
							Amount:       decmailAmount,
						}
						if err := tx.Create(&stakingAction).Error; err != nil {
							return err
						}
					}
				case GovernaceCacelHash:
					from, err := getAddresFromHash256(topics[1])
					if err != nil {
						return err
					}
					to, err := getForwardToAddressByFrom(from.String())
					if err != nil {
						return err
					}
					bucketIDs, err := getBucketIDsByAddressWithHeight(to, blk.Height())
					if err != nil {
						return err
					}
					for _, bucketID := range bucketIDs {
						decmailAmount, err := getBucketSumAmountByBucketID(tx, bucketID)
						if err != nil {
							return errors.Wrapf(err, "getBucketSumAmountByBucketID error, bucketID: %d", bucketID)
						}
						info, err := getBucketInfoAddressByBucketID(tx, bucketID)
						if err != nil {
							return errors.Wrap(err, "getBucketInfoAddressByBucketID error")
						}
						if info.OwnerAddress != from.String() {
							continue
						}
						stakingAction = models.StakingAction{
							BlockHeight:  blk.Height(),
							BucketID:     bucketID,
							OwnerAddress: to,
							Sender:       sender.String(),
							ActHash:      actHash,
							ForwardTo:    from.String(),
							Candidate:    info.Candidate,
							AutoStake:    info.AutoStake,
							ActType:      "GovernaceCacel",
							Duration:     info.Duration,
							Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
						}
						if err := tx.Create(&stakingAction).Error; err != nil {
							return err
						}
						stakingAction = models.StakingAction{
							BlockHeight:  blk.Height(),
							BucketID:     bucketID,
							OwnerAddress: info.OwnerAddress,
							Sender:       sender.String(),
							ActHash:      actHash,
							Candidate:    info.Candidate,
							AutoStake:    info.AutoStake,
							ActType:      "GovernaceCacel",
							Duration:     info.Duration,
							Amount:       decmailAmount,
						}
						if err := tx.Create(&stakingAction).Error; err != nil {
							return err
						}
					}
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
