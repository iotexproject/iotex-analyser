package staking_bucket

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.2.1"

const (
	StakingProtocolAddress = "io1qnpz47hx5q6r3w876axtrn6yz95d70cjl35r53"
)

var unSelfStake *big.Int

type StakingBucketPlugin struct {
	plugin.PluginShadow
	fixingStakeAmount bool
}

func (b StakingBucketPlugin) Name() string {
	return b.ShadowName("staking_bucket")
}

func (b StakingBucketPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b StakingBucketPlugin) DependentPlugins() []string {
	return []string{"candidate", "slash"}
}

func (b StakingBucketPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), b.ShadowTable(&models.StakingBucket{})); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	var ok bool
	unSelfStake, ok = new(big.Int).SetString("000000000000000000000000000000000000000000000000ffffffffffffffff", 16)
	if !ok {
		return errors.New("can not convert string to bigint with plugin %s:" + b.Name())
	}

	return nil
}

func (b StakingBucketPlugin) fixStakeAmount(ctx context.Context) error {
	db := db.DB()
	query := fmt.Sprintf("select * from %s where act_type='Unstake' order by id asc", b.ShadowTable(&models.StakingBucket{}).TableName())
	rows, err := db.Raw(query).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var bucket models.StakingBucket
		if err := db.ScanRows(rows, b.ShadowTable(&bucket)); err != nil {
			return err
		}
		decmailAmount, err := b.getBucketSumAmountByBucketID(db, bucket.BucketID)
		if err != nil {
			return err
		}
		if err := db.Model(b.ShadowTable(&bucket)).Update("staked_amount", decmailAmount).Error; err != nil {
			return err
		}
	}
	return nil
}

func (b StakingBucketPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if !b.fixingStakeAmount {
		store := &db.Store{
			Key: b.ShadowName("staking_bucket_fix_stake_amount"),
		}
		if err := store.Get(); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				//it means the data is not fixed
				log.L().Info("staking_bucket fixing stake amount")
				if err := b.fixStakeAmount(ctx); err != nil {
					return err
				}
				store.Value = fmt.Sprintf(`{"block_height": %d}`, blk.Height())
				if err := store.Save(); err != nil {
					return err
				}
			} else {
				return err
			}
		}
		b.fixingStakeAmount = true
	}
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if err := b.handleBlock(ctx, blk, tx); err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return err
}

func (b StakingBucketPlugin) BatchSize() int {
	return 0
}

func (b StakingBucketPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, blk := range blks {
			if err := b.handleBlock(ctx, blk, tx); err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blks[len(blks)-1].Height())
	})
	return err
}

func (b StakingBucketPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b StakingBucketPlugin) Version() string {
	return VERSION
}

func (b StakingBucketPlugin) handleBlock(ctx context.Context, blk *block.Block, tx *gorm.DB) error {
	var stakingBucket models.StakingBucket
	actions := make(map[hash.Hash256]*action.SealedEnvelope, len(blk.Actions))
	bucketMap := make(map[string]uint64)
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
		act := selp.Action()
		actionHash, _ := selp.Hash()
		actHash := hex.EncodeToString(actionHash[:])
		// cmpNum := big.NewInt(100000000)
		for _, log := range receipt.Logs() {
			if log.Address == StakingProtocolAddress && len(log.Topics) > 1 {
				bucketIndex := new(big.Int).SetBytes(log.Topics[1][:])

				// if bucketIndex.Cmp(cmpNum) > 0 {
				// 	continue
				// }
				bucketMap[actHash] = bucketIndex.Uint64()
			}
		}
		switch a := act.(type) {
		case *action.WithdrawStake:
			bucketID := a.BucketIndex()
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return err
			}
			stakingBucket = models.StakingBucket{
				BlockHeight:      blk.Height(),
				BucketID:         bucketID,
				CreateTime:       info.CreateTime,
				StakeStartTime:   info.StakeStartTime,
				UnstakeStartTime: info.UnstakeStartTime,
				StakedAmount:     decimal.NewFromInt(0),
				VotingPower:      decimal.NewFromInt(0),
				OwnerAddress:     info.OwnerAddress,
				Sender:           sender.String(),
				ActionHash:       actHash,
				Candidate:        info.Candidate,
				AutoStake:        false,
				ActType:          "WithdrawStake",
				Duration:         0,
				Amount:           decimal.NewFromInt(0),
				Timestamp:        blk.Timestamp().Unix(),
			}
			if err := tx.Create(b.ShadowTable(&stakingBucket)).Error; err != nil {
				return err
			}
		case *action.CreateStake:
			cadidateAddr, err := getCandidateAddressByName(a.Candidate(), blk.Height())
			if err != nil {
				return err
			}

			bucketID, ok := bucketMap[actHash]
			if !ok {
				return errors.New("can not found bucketID with actHash:" + actHash)
			}

			voteWeight := getVoteWeight(a.Duration(), a.Amount(), a.AutoStake(), sender.String() == cadidateAddr)
			stakingBucket = models.StakingBucket{
				BlockHeight:      blk.Height(),
				BucketID:         bucketID,
				CreateTime:       blk.Timestamp().Unix(),
				StakeStartTime:   blk.Timestamp().Unix(),
				UnstakeStartTime: 0,
				StakedAmount:     decimal.NewFromBigInt(a.Amount(), 0),
				VotingPower:      decimal.NewFromBigInt(voteWeight, 0),
				Sender:           sender.String(),
				OwnerAddress:     sender.String(),
				ActionHash:       actHash,
				Candidate:        cadidateAddr,
				Amount:           decimal.NewFromBigInt(a.Amount(), 0),
				ActType:          "StakeCreate",
				AutoStake:        a.AutoStake(),
				Duration:         a.Duration(),
				Timestamp:        blk.Timestamp().Unix(),
			}
			if err := tx.Create(b.ShadowTable(&stakingBucket)).Error; err != nil {
				return err
			}
		case *action.TransferStake:
			bucketID := a.BucketIndex()
			decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketID)
			if err != nil {
				return err
			}
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return err
			}
			voteWeight := getVoteWeight(info.Duration, decmailAmount.Coefficient(), info.AutoStake, a.VoterAddress().String() == info.Candidate)
			stakingBucket = models.StakingBucket{
				BlockHeight:      blk.Height(),
				BucketID:         bucketID,
				CreateTime:       info.CreateTime,
				StakeStartTime:   info.StakeStartTime,
				UnstakeStartTime: info.UnstakeStartTime,
				StakedAmount:     decmailAmount,
				VotingPower:      decimal.NewFromBigInt(voteWeight, 0),
				OwnerAddress:     a.VoterAddress().String(),
				Sender:           sender.String(),
				ActionHash:       actHash,
				Candidate:        info.Candidate,
				AutoStake:        info.AutoStake,
				ActType:          "TransferStake",
				Duration:         info.Duration,
				Amount:           decimal.NewFromInt(0),
				Timestamp:        blk.Timestamp().Unix(),
			}
			if err := tx.Create(b.ShadowTable(&stakingBucket)).Error; err != nil {
				return err
			}
		case *action.Restake:
			bucketID := a.BucketIndex()
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return err
			}
			// fix greenland (height=6544441) restake
			fixAmount := decimal.NewFromInt(0)
			if blk.Height() < genesis.Default.GreenlandBlockHeight {
				fixAmount, err = b.getFixBucketSumAmountByBucketID(tx, bucketID)
				if err != nil {
					return err
				}
			}
			decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketID)
			if err != nil {
				return err
			}
			voteWeight := getVoteWeight(a.Duration(), decmailAmount.Coefficient(), a.AutoStake(), sender.String() == info.Candidate)
			stakingBucket = models.StakingBucket{
				BlockHeight:      blk.Height(),
				BucketID:         bucketID,
				CreateTime:       info.CreateTime,
				StakeStartTime:   blk.Timestamp().Unix(),
				UnstakeStartTime: info.UnstakeStartTime,
				StakedAmount:     decmailAmount,
				VotingPower:      decimal.NewFromBigInt(voteWeight, 0),
				OwnerAddress:     sender.String(),
				Sender:           sender.String(),
				ActionHash:       actHash,
				Candidate:        info.Candidate,
				AutoStake:        a.AutoStake(),
				ActType:          "Restake",
				Duration:         a.Duration(),
				Amount:           fixAmount,
				Timestamp:        blk.Timestamp().Unix(),
			}
			if err := tx.Create(b.ShadowTable(&stakingBucket)).Error; err != nil {
				return err
			}
		case *action.ChangeCandidate:
			bucketID := a.BucketIndex()
			decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketID)
			if err != nil {
				return err
			}
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return err
			}
			cadidateAddr, err := getCandidateAddressByName(a.Candidate(), blk.Height())
			if err != nil {
				return err
			}
			voteWeight := getVoteWeight(info.Duration, decmailAmount.Coefficient(), info.AutoStake, info.OwnerAddress == cadidateAddr)
			stakingBucket = models.StakingBucket{
				BlockHeight:      blk.Height(),
				BucketID:         bucketID,
				CreateTime:       info.CreateTime,
				StakeStartTime:   info.StakeStartTime,
				UnstakeStartTime: info.UnstakeStartTime,
				StakedAmount:     decmailAmount,
				VotingPower:      decimal.NewFromBigInt(voteWeight, 0),
				OwnerAddress:     info.OwnerAddress,
				Sender:           sender.String(),
				ActionHash:       actHash,
				Candidate:        cadidateAddr,
				AutoStake:        info.AutoStake,
				ActType:          "ChangeCandidate",
				Duration:         info.Duration,
				Amount:           decimal.NewFromInt(0),
				Timestamp:        blk.Timestamp().Unix(),
			}
			if err := tx.Create(b.ShadowTable(&stakingBucket)).Error; err != nil {
				return err
			}
		case *action.DepositToStake:
			bucketID := a.BucketIndex()
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return err
			}
			decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketID)
			if err != nil {
				return err
			}
			stakedAmount := decmailAmount.Add(decimal.NewFromBigInt(a.Amount(), 0))
			voteWeight := getVoteWeight(info.Duration, stakedAmount.Coefficient(), info.AutoStake, info.OwnerAddress == info.Candidate)

			stakingBucket = models.StakingBucket{
				BlockHeight:      blk.Height(),
				BucketID:         bucketID,
				CreateTime:       info.CreateTime,
				StakeStartTime:   info.StakeStartTime,
				UnstakeStartTime: info.UnstakeStartTime,
				StakedAmount:     stakedAmount,
				VotingPower:      decimal.NewFromBigInt(voteWeight, 0),
				OwnerAddress:     info.OwnerAddress,
				Sender:           sender.String(),
				ActionHash:       actHash,
				Candidate:        info.Candidate,
				AutoStake:        info.AutoStake,
				ActType:          "DepositToStake",
				Duration:         info.Duration,
				Amount:           decimal.NewFromBigInt(a.Amount(), 0),
				Timestamp:        blk.Timestamp().Unix(),
			}
			if err := tx.Create(b.ShadowTable(&stakingBucket)).Error; err != nil {
				return err
			}
		case *action.Unstake:
			bucketID := a.BucketIndex()
			decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketID)
			if err != nil {
				return err
			}
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return err
			}
			stakingBucket = models.StakingBucket{
				BlockHeight:      blk.Height(),
				BucketID:         bucketID,
				CreateTime:       info.CreateTime,
				StakeStartTime:   info.StakeStartTime,
				UnstakeStartTime: blk.Timestamp().Unix(),
				StakedAmount:     decmailAmount,
				VotingPower:      decimal.NewFromInt(0),
				OwnerAddress:     info.OwnerAddress,
				Sender:           sender.String(),
				ActionHash:       actHash,
				Candidate:        info.Candidate,
				AutoStake:        info.AutoStake,
				ActType:          "Unstake",
				Duration:         info.Duration,
				Amount:           decimal.NewFromInt(0),
				Timestamp:        blk.Timestamp().Unix(),
			}
			if err := tx.Create(b.ShadowTable(&stakingBucket)).Error; err != nil {
				return err
			}
		case *action.CandidateRegister:
			bucketID, ok := bucketMap[actHash]
			if !ok {
				return errors.New("can not found bucketID with actHash:" + actHash)
			}

			if bucketID != unSelfStake.Uint64() {
				stakedAmount := decimal.NewFromBigInt(a.Amount(), 0)
				voteWeight := getVoteWeight(a.Duration(), a.Amount(), a.AutoStake(), true)

				stakingBucket = models.StakingBucket{
					BlockHeight:  blk.Height(),
					BucketID:     bucketID,
					CreateTime:   blk.Timestamp().Unix(),
					StakedAmount: stakedAmount,
					VotingPower:  decimal.NewFromBigInt(voteWeight, 0),
					Sender:       sender.String(),
					OwnerAddress: a.OwnerAddress().String(),
					ActionHash:   actHash,
					Candidate:    a.OwnerAddress().String(),
					Amount:       decimal.NewFromBigInt(a.Amount(), 0),
					ActType:      "CandidateRegister",
					AutoStake:    a.AutoStake(),
					Duration:     a.Duration(),
					Timestamp:    blk.Timestamp().Unix(),
				}
				if err := tx.Create(b.ShadowTable(&stakingBucket)).Error; err != nil {
					return err
				}
			}
		case *action.MigrateStake:
			bucketID := a.BucketIndex()
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return err
			}
			stakingBucket = models.StakingBucket{
				BlockHeight:      blk.Height(),
				BucketID:         bucketID,
				CreateTime:       info.CreateTime,
				StakeStartTime:   info.StakeStartTime,
				UnstakeStartTime: info.UnstakeStartTime,
				StakedAmount:     decimal.NewFromInt(0),
				VotingPower:      decimal.NewFromInt(0),
				OwnerAddress:     info.OwnerAddress,
				Sender:           sender.String(),
				ActionHash:       actHash,
				Candidate:        info.Candidate,
				AutoStake:        false,
				ActType:          "MigrateStake",
				Duration:         0,
				Amount:           decimal.NewFromInt(0),
				Timestamp:        blk.Timestamp().Unix(),
			}
			if err := tx.Create(b.ShadowTable(&stakingBucket)).Error; err != nil {
				return err
			}
		case *action.CandidateEndorsement:
			bucketID := a.BucketIndex()
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return errors.Wrapf(err, "failed to get bucket info by bucketID %d", bucketID)
			}
			bucket, err := GetStakingBucketByID(bucketID, blk.Height())
			if err != nil {
				return errors.Wrap(err, "failed to get staking bucket")
			}
			stakeAmount, err := decimal.NewFromString(bucket.StakedAmount)
			if err != nil {
				return errors.Wrap(err, "failed to parse staked amount")
			}
			selfStake := false
			if a.Op() == action.CandidateEndorsementOpEndorse {
				selfStake = true
			}
			votes := getVoteWeight(info.Duration, stakeAmount.BigInt(), bucket.AutoStake, selfStake)
			stakingBucket = models.StakingBucket{
				BlockHeight:             blk.Height(),
				BucketID:                bucketID,
				CreateTime:              info.CreateTime,
				StakeStartTime:          info.StakeStartTime,
				UnstakeStartTime:        info.UnstakeStartTime,
				StakedAmount:            stakeAmount,
				VotingPower:             decimal.NewFromBigInt(votes, 0),
				OwnerAddress:            info.OwnerAddress,
				Sender:                  sender.String(),
				ActionHash:              actHash,
				Candidate:               info.Candidate,
				AutoStake:               info.AutoStake,
				ActType:                 "CandidateEndorsement",
				EndorsementExpireHeight: bucket.EndorsementExpireBlockHeight,
				Duration:                0,
				Amount:                  decimal.NewFromInt(0),
				Timestamp:               blk.Timestamp().Unix(),
			}
			if err := tx.Create(b.ShadowTable(&stakingBucket)).Error; err != nil {
				return err
			}
		case *action.GrantReward:
			if a.RewardType() != action.EpochReward {
				continue
			}
			slashs, err := models.FetchSlashByActionHash(actHash, tx)
			if err != nil {
				return errors.Wrapf(err, "failed to fetch slash by action hash %s", actHash)
			}
			for _, slash := range slashs {
				bucketID := slash.BucketID
				info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
				if err != nil {
					return errors.Wrapf(err, "failed to get bucket info by bucketID %d", bucketID)
				}
				decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketID)
				if err != nil {
					return err
				}
				stakedAmount := decmailAmount.Add(slash.Amount.Neg())
				voteWeight := getVoteWeight(info.Duration, stakedAmount.Coefficient(), info.AutoStake, true)

				stakingBucket = models.StakingBucket{
					BlockHeight:      blk.Height(),
					BucketID:         bucketID,
					CreateTime:       info.CreateTime,
					StakeStartTime:   info.StakeStartTime,
					UnstakeStartTime: info.UnstakeStartTime,
					StakedAmount:     stakedAmount,
					VotingPower:      decimal.NewFromBigInt(voteWeight, 0),
					OwnerAddress:     info.OwnerAddress,
					Sender:           sender.String(),
					ActionHash:       actHash,
					Candidate:        info.Candidate,
					AutoStake:        info.AutoStake,
					ActType:          "SlashCandidate",
					Duration:         info.Duration,
					Amount:           slash.Amount.Neg(),
					Timestamp:        blk.Timestamp().Unix(),
				}
				if err := tx.Create(b.ShadowTable(&stakingBucket)).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}
