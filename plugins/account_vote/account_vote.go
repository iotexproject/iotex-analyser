package main

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.2.1"

const (
	GovernaceForwardAddress = "io1xfdn0z046hzm03jrtm8hf4scw2w07t7a0mqtmz"
	GovernaceForwardToHash  = "87f136fb149a2751c6cc2ee9d9d2f7fc786ac7a163f58ef9fdbe2f1d7e7c78e8"
	GovernaceCacelHash      = "dfae2e44eee3429afab9409ee9f946d11d84e8eee5d3c81525197a2925b0ceb9"
)

type accountVotePlugin struct {
}

func (b accountVotePlugin) Name() string {
	return "account_vote"
}

func (b accountVotePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b accountVotePlugin) Start(ctx context.Context) error {
	if err := db.DB().AutoMigrate(&AccountVote{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	var err error
	config, _ := kernel.GetConfigCtx(ctx)
	_, err = newConfig(config)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s plugin config", b.Name())
	}
	return nil
}

func (b accountVotePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		var accountVote AccountVote
		actions := make(map[hash.Hash256]action.SealedEnvelope, len(blk.Actions))
		for _, selp := range blk.Actions {
			actions[selp.Hash()] = selp
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
			switch a := act.(type) {
			case *action.CreateStake:
				cadidateAddr, err := getCandidateAddressByName(a.Candidate())
				if err != nil {
					return err
				}
				actionHash := selp.Hash()
				bucketID, err := getBucketIDByActHash(hex.EncodeToString(actionHash[:]))
				if err != nil {
					return err
				}
				accountVote = AccountVote{
					BlockHeight: blk.Height(),
					BucketID:    bucketID,
					Address:     sender.String(),
					Candidate:   cadidateAddr,
					Amount:      decimal.NewFromBigInt(a.Amount(), 0),
					ActType:     "StakeCreate",
					AutoStake:   a.AutoStake(),
					Duration:    a.Duration(),
				}
				if err := tx.Create(&accountVote).Error; err != nil {
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
				accountVote = AccountVote{
					BlockHeight: blk.Height(),
					BucketID:    bucketID,
					Address:     info.Address,
					Candidate:   info.Candidate,
					AutoStake:   info.AutoStake,
					ActType:     "TransferStake",
					Duration:    info.Duration,
					Amount:      decmailAmount.Mul(decimal.NewFromInt(-1)),
				}
				if err := tx.Create(&accountVote).Error; err != nil {
					return err
				}
				accountVote = AccountVote{
					BlockHeight: blk.Height(),
					BucketID:    bucketID,
					Address:     a.VoterAddress().String(),
					Candidate:   info.Candidate,
					AutoStake:   info.AutoStake,
					ActType:     "TransferStake",
					Duration:    info.Duration,
					Amount:      decmailAmount,
				}
				if err := tx.Create(&accountVote).Error; err != nil {
					return err
				}
			case *action.Restake:
				bucketID := a.BucketIndex()
				info, err := getBucketInfoAddressByBucketID(tx, bucketID)
				if err != nil {
					return errors.Wrap(err, "getBucketInfoAddressByBucketID error")
				}
				accountVote = AccountVote{
					BlockHeight: blk.Height(),
					BucketID:    bucketID,
					Address:     sender.String(),
					Candidate:   info.Candidate,
					AutoStake:   a.AutoStake(),
					ActType:     "Restake",
					Duration:    a.Duration(),
					Amount:      decimal.NewFromInt(0),
				}
				if err := tx.Create(&accountVote).Error; err != nil {
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
				accountVote = AccountVote{
					BlockHeight: blk.Height(),
					BucketID:    bucketID,
					Address:     info.Address,
					Candidate:   info.Candidate,
					AutoStake:   info.AutoStake,
					ActType:     "ChangeCandidate",
					Duration:    info.Duration,
					Amount:      decmailAmount.Mul(decimal.NewFromInt(-1)),
				}
				if err := tx.Create(&accountVote).Error; err != nil {
					return err
				}
				cadidateAddr, err := getCandidateAddressByName(a.Candidate())
				if err != nil {
					return err
				}
				accountVote = AccountVote{
					BlockHeight: blk.Height(),
					BucketID:    bucketID,
					Address:     info.Address,
					Candidate:   cadidateAddr,
					AutoStake:   info.AutoStake,
					ActType:     "ChangeCandidate",
					Duration:    info.Duration,
					Amount:      decmailAmount,
				}
				if err := tx.Create(&accountVote).Error; err != nil {
					return err
				}
			case *action.DepositToStake:
				bucketID := a.BucketIndex()
				info, err := getBucketInfoAddressByBucketID(tx, bucketID)
				if err != nil {
					return errors.Wrap(err, "getBucketInfoAddressByBucketID error")
				}
				accountVote = AccountVote{
					BlockHeight: blk.Height(),
					BucketID:    bucketID,
					Address:     sender.String(),
					Candidate:   info.Candidate,
					AutoStake:   info.AutoStake,
					ActType:     "DepositToStake",
					Duration:    info.Duration,
					Amount:      decimal.NewFromBigInt(a.Amount(), 0),
				}
				if err := tx.Create(&accountVote).Error; err != nil {
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
				accountVote = AccountVote{
					BlockHeight: blk.Height(),
					BucketID:    bucketID,
					Address:     sender.String(),
					Candidate:   info.Candidate,
					AutoStake:   info.AutoStake,
					ActType:     "Unstake",
					Duration:    info.Duration,
					Amount:      decmailAmount.Mul(decimal.NewFromInt(-1)),
				}
				if err := tx.Create(&accountVote).Error; err != nil {
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
						accountVote = AccountVote{
							BlockHeight: blk.Height(),
							BucketID:    bucketID,
							Address:     from.String(),
							ForwardTo:   to.String(),
							Candidate:   info.Candidate,
							AutoStake:   info.AutoStake,
							ActType:     "GovernaceForward",
							Duration:    info.Duration,
							Amount:      decmailAmount.Mul(decimal.NewFromInt(-1)),
						}
						if err := tx.Create(&accountVote).Error; err != nil {
							return err
						}
						accountVote = AccountVote{
							BlockHeight: blk.Height(),
							BucketID:    bucketID,
							Address:     to.String(),
							Candidate:   info.Candidate,
							AutoStake:   info.AutoStake,
							ActType:     "GovernaceForward",
							Duration:    info.Duration,
							Amount:      decmailAmount,
						}
						if err := tx.Create(&accountVote).Error; err != nil {
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
						accountVote = AccountVote{
							BlockHeight: blk.Height(),
							BucketID:    bucketID,
							Address:     to,
							ForwardTo:   from.String(),
							Candidate:   info.Candidate,
							AutoStake:   info.AutoStake,
							ActType:     "GovernaceCacel",
							Duration:    info.Duration,
							Amount:      decmailAmount.Mul(decimal.NewFromInt(-1)),
						}
						if err := tx.Create(&accountVote).Error; err != nil {
							return err
						}
						accountVote = AccountVote{
							BlockHeight: blk.Height(),
							BucketID:    bucketID,
							Address:     from.String(),
							Candidate:   info.Candidate,
							AutoStake:   info.AutoStake,
							ActType:     "GovernaceCacel",
							Duration:    info.Duration,
							Amount:      decmailAmount,
						}
						if err := tx.Create(&accountVote).Error; err != nil {
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

func (b accountVotePlugin) Stop(ctx context.Context) error {
	return nil
}

func (b accountVotePlugin) Version() string {
	return VERSION
}

// exported
var Plugin = accountVotePlugin{}
