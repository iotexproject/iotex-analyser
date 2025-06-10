package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/kernel/systemstaking"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

const VERSION = "2.2.5"

const (
	errBucketSumAmount             = "getBucketSumAmountByBucketID error, bucketID: %d"
	errBucketInfoAddressByBucketID = "getBucketInfoAddressByBucketID error"
)

type Config struct {
	Height          uint64 `yaml:"height"`
	ContractAddress string `yaml:"contractAddress"`
}

type systemStakingBucketPlugin struct {
	startHeight     uint64
	contractAddress string
}

func (b systemStakingBucketPlugin) Name() string {
	return "system_staking_bucket_v3_record"
}

func (b systemStakingBucketPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *systemStakingBucketPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.SystemStakingBucketV3Record{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	height, err := db.GetIndexHeight(b.Name())
	if err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	cfg := &Config{}
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		if err = yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		}
		log.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		b.startHeight = cfg.Height
		b.contractAddress = cfg.ContractAddress
	} else {
		b.startHeight = config.Default.Genesis.WakeBlockHeight
		b.contractAddress = config.Default.Genesis.Poll.SystemStakingContractV3Address
	}
	if height == 0 {
		return db.UpdateIndexHeight(b.Name(), b.startHeight)
	}
	return nil
}

func (b systemStakingBucketPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		var stakingBucket models.SystemStakingBucketV3Record
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
			actionHash, _ := selp.Hash()
			actHash := hex.EncodeToString(actionHash[:])
			zeroAmount := decimal.NewFromInt(0)
			for _, log := range receipt.Logs() {
				if log.Address != b.contractAddress {
					continue
				}
				abiEvent, err := systemstaking.ABI.EventByID(common.Hash(log.Topics[0]))
				if err != nil {
					return errors.WithStack(err)
				}
				switch abiEvent.Name {
				case "Staked": // Staked(uint256 indexed bucketId, address delegate, uint256 amount, uint256 duration)
					bucketId := new(big.Int).SetBytes(log.Topics[1][:])
					info, err := b.getBucketInfoAddressByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return err
					}
					event := struct {
						Delegate common.Address
						Amount   *big.Int
						Duration *big.Int
					}{}
					err = systemstaking.ABI.UnpackIntoInterface(&event, "Staked", log.Data)
					if err != nil {
						return errors.WithStack(err)
					}
					decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return err
					}
					stakedAmount := decmailAmount.Add(decimal.NewFromBigInt(event.Amount, 0))
					duration := time.Duration(event.Duration.Uint64()) * time.Second
					voteWeight := systemstaking.GetVoteWeight(blk.Height(), duration, stakedAmount.Coefficient(), true, false)
					cadidateAddr, _ := address.FromHex(event.Delegate.String())
					stakingBucket = models.SystemStakingBucketV3Record{
						SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
							BlockHeight:          blk.Height(),
							BucketID:             bucketId.Uint64(),
							Sender:               sender.String(),
							OwnerAddress:         info.OwnerAddress,
							ActHash:              actHash,
							CreateTime:           info.CreateTime,
							StakeStartTime:       blk.Timestamp().Unix(),
							StakedAmount:         stakedAmount,
							VotingPower:          decimal.NewFromBigInt(voteWeight, 0),
							UnstakeStartTime:     0,
							DelegateOwnerAddress: cadidateAddr.String(),
							Amount:               decimal.NewFromBigInt(event.Amount, 0),
							EventType:            "Staked",
							AutoStake:            true, //default true
							Duration:             uint32(duration.Seconds()),
							DurationType:         2, // seconds in days
							Timestamp:            blk.Timestamp().Unix(),
							Final:                true,
						},
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "Transfer":
					from, _ := address.FromBytes(log.Topics[1][:])
					to, _ := address.FromBytes(log.Topics[2][:])
					bucketId := new(big.Int).SetBytes(log.Topics[3][:])
					if from.String() == address.ZeroAddress {
						//mint skip
						stakingBucket = models.SystemStakingBucketV3Record{
							SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
								BlockHeight:          blk.Height(),
								BucketID:             bucketId.Uint64(),
								OwnerAddress:         to.String(),
								Sender:               sender.String(),
								ActHash:              actHash,
								CreateTime:           blk.Timestamp().Unix(),
								StakeStartTime:       0,
								UnstakeStartTime:     0,
								DelegateOwnerAddress: "",
								AutoStake:            false,
								EventType:            "Transfer",
								Duration:             0,
								DurationType:         2,
								Amount:               decimal.NewFromInt(0),
								Timestamp:            blk.Timestamp().Unix(),
								Final:                false,
							},
						}
					} else {
						info, err := b.getBucketInfoAddressByBucketID(tx, bucketId.Uint64())
						if err != nil {
							return errors.Wrap(err, errBucketInfoAddressByBucketID)
						}
						stakeAmount, _ := decimal.NewFromString(info.StakedAmount)
						votingPower, _ := decimal.NewFromString(info.VotingPower)
						stakingBucket = models.SystemStakingBucketV3Record{
							SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
								BlockHeight:          blk.Height(),
								BucketID:             bucketId.Uint64(),
								OwnerAddress:         to.String(),
								Sender:               sender.String(),
								ActHash:              actHash,
								CreateTime:           info.CreateTime,
								StakeStartTime:       info.StakeStartTime,
								UnstakeStartTime:     info.UnstakeStartTime,
								StakedAmount:         stakeAmount,
								VotingPower:          votingPower,
								DelegateOwnerAddress: info.DelegateOwnerAddress,
								AutoStake:            info.AutoStake,
								EventType:            "Transfer",
								Duration:             info.Duration,
								DurationType:         info.DurationType,
								Amount:               zeroAmount,
								Timestamp:            blk.Timestamp().Unix(),
								Final:                true,
							},
						}
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}

				case "Unstaked": // Unstaked(uint256 indexed bucketId)
					bucketId := new(big.Int).SetBytes(log.Topics[1][:])
					decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, bucketId)
					}
					info, err := b.getBucketInfoAddressByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}

					stakingBucket = models.SystemStakingBucketV3Record{
						SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
							BlockHeight:          blk.Height(),
							BucketID:             bucketId.Uint64(),
							OwnerAddress:         info.OwnerAddress,
							Sender:               sender.String(),
							ActHash:              actHash,
							CreateTime:           info.CreateTime,
							StakeStartTime:       info.StakeStartTime,
							UnstakeStartTime:     blk.Timestamp().Unix(),
							StakedAmount:         zeroAmount,
							VotingPower:          zeroAmount,
							DelegateOwnerAddress: info.DelegateOwnerAddress,
							AutoStake:            info.AutoStake,
							EventType:            "Unstaked",
							Duration:             info.Duration,
							DurationType:         info.DurationType,
							Amount:               decmailAmount.Mul(decimal.NewFromInt(-1)),
							Timestamp:            blk.Timestamp().Unix(),
							Final:                true,
							Muted:                info.Muted,
						},
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "Locked": // Locked(uint256 indexed bucketId, uint256 duration)
					bucketId := new(big.Int).SetBytes(log.Topics[1][:])
					event := struct {
						Duration *big.Int
					}{}
					err := systemstaking.ABI.UnpackIntoInterface(&event, "Locked", log.Data)
					if err != nil {
						return errors.WithStack(err)
					}
					info, err := b.getBucketInfoAddressByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, bucketId)
					}
					duration := time.Duration(event.Duration.Uint64()) * time.Second
					autoStake := true //locked must auto stake
					voteWeight := systemstaking.GetVoteWeight(blk.Height(), duration, decmailAmount.BigInt(), autoStake, false)
					stakingBucket = models.SystemStakingBucketV3Record{
						SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
							BlockHeight:          blk.Height(),
							BucketID:             bucketId.Uint64(),
							Sender:               sender.String(),
							OwnerAddress:         info.OwnerAddress,
							ActHash:              actHash,
							CreateTime:           info.CreateTime,
							StakeStartTime:       info.StakeStartTime,
							UnstakeStartTime:     info.UnstakeStartTime,
							StakedAmount:         decmailAmount,
							VotingPower:          decimal.NewFromBigInt(voteWeight, 0),
							DelegateOwnerAddress: info.DelegateOwnerAddress,
							Amount:               zeroAmount,
							EventType:            "Locked",
							AutoStake:            autoStake,
							Duration:             uint32(duration.Seconds()),
							DurationType:         info.DurationType,
							Timestamp:            blk.Timestamp().Unix(),
							Final:                true,
							Muted:                info.Muted,
						},
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "Unlocked": // Unlocked(uint256 indexed bucketId)
					bucketId := new(big.Int).SetBytes(log.Topics[1][:])
					info, err := b.getBucketInfoAddressByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, bucketId)
					}
					duration := time.Duration(info.Duration) * time.Second
					autoStake := false //unlocked means auto stake is false
					voteWeight := systemstaking.GetVoteWeight(blk.Height(), duration, decmailAmount.BigInt(), autoStake, false)
					stakingBucket = models.SystemStakingBucketV3Record{
						SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
							BlockHeight:          blk.Height(),
							BucketID:             bucketId.Uint64(),
							Sender:               sender.String(),
							OwnerAddress:         info.OwnerAddress,
							ActHash:              actHash,
							CreateTime:           info.CreateTime,
							StakeStartTime:       info.StakeStartTime,
							UnstakeStartTime:     info.UnstakeStartTime,
							StakedAmount:         decmailAmount,
							VotingPower:          decimal.NewFromBigInt(voteWeight, 0),
							DelegateOwnerAddress: info.DelegateOwnerAddress,
							Amount:               zeroAmount,
							EventType:            "Unlocked",
							AutoStake:            false,
							Duration:             info.Duration,
							DurationType:         info.DurationType,
							Timestamp:            blk.Timestamp().Unix(),
							Final:                true,
							Muted:                info.Muted,
						},
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "Merged": // Merged(uint256[] bucketIds, uint256 amount, uint256 duration)
					event := struct {
						bucketIds []*big.Int
						Amount    *big.Int
						Duration  *big.Int
					}{}
					eventMap := make(map[string]interface{})
					err := systemstaking.ABI.UnpackIntoMap(eventMap, "Merged", log.Data)
					if err != nil {
						return errors.WithStack(err)
					}
					event.bucketIds, ok = eventMap["bucketIds"].([]*big.Int)
					if !ok {
						return errors.New("bucketIds type error")
					}
					event.Amount, ok = eventMap["amount"].(*big.Int)
					if !ok {
						return errors.New("amount type error")
					}
					event.Duration, ok = eventMap["duration"].(*big.Int)
					if !ok {
						return errors.New("duration type error")
					}
					decmailAmount := decimal.NewFromBigInt(event.Amount, 0)
					//Here the amount of tokenIDs from the second tokenID are added to the first tokenID
					for _, tokenID := range event.bucketIds[1:] {
						tokenAmount, err := b.getBucketSumAmountByBucketID(tx, tokenID.Uint64())
						if err != nil {
							return errors.Wrapf(err, errBucketSumAmount, tokenID)
						}
						info, err := b.getBucketInfoAddressByBucketID(tx, tokenID.Uint64())
						if err != nil {
							return errors.Wrap(err, errBucketInfoAddressByBucketID)
						}
						stakingBucket = models.SystemStakingBucketV3Record{
							SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
								BlockHeight:          blk.Height(),
								BucketID:             tokenID.Uint64(),
								Sender:               sender.String(),
								OwnerAddress:         info.OwnerAddress,
								ActHash:              actHash,
								CreateTime:           info.CreateTime,
								StakeStartTime:       info.StakeStartTime,
								UnstakeStartTime:     info.UnstakeStartTime,
								StakedAmount:         zeroAmount,
								VotingPower:          zeroAmount,
								DelegateOwnerAddress: info.DelegateOwnerAddress,
								Amount:               tokenAmount.Mul(decimal.NewFromInt(-1)),
								EventType:            "Merged",
								AutoStake:            info.AutoStake,
								Duration:             info.Duration,
								DurationType:         info.DurationType,
								Timestamp:            blk.Timestamp().Unix(),
								Final:                true,
							},
						}
						if err := tx.Create(&stakingBucket).Error; err != nil {
							return err
						}
					}
					bucketId := event.bucketIds[0]
					tokenAmount, err := b.getBucketSumAmountByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, bucketId)
					}
					info, err := b.getBucketInfoAddressByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					duration := time.Duration(event.Duration.Uint64()) * time.Second
					autoStake := true
					stakeAmount := decmailAmount
					voteWeight := systemstaking.GetVoteWeight(blk.Height(), duration, stakeAmount.BigInt(), autoStake, false)
					stakingBucket = models.SystemStakingBucketV3Record{
						SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
							BlockHeight:          blk.Height(),
							BucketID:             bucketId.Uint64(),
							Sender:               sender.String(),
							OwnerAddress:         info.OwnerAddress,
							ActHash:              actHash,
							CreateTime:           info.CreateTime,
							StakeStartTime:       info.StakeStartTime,
							UnstakeStartTime:     info.UnstakeStartTime,
							StakedAmount:         stakeAmount,
							VotingPower:          decimal.NewFromBigInt(voteWeight, 0),
							DelegateOwnerAddress: info.DelegateOwnerAddress,
							Amount:               decmailAmount.Sub(tokenAmount),
							EventType:            "Merged",
							AutoStake:            autoStake,
							Duration:             uint32(duration.Seconds()),
							DurationType:         info.DurationType,
							Timestamp:            blk.Timestamp().Unix(),
							Final:                true,
						},
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "BucketExpanded": // BucketExpanded(uint256 indexed bucketId, uint256 amount, uint256 duration)
					bucketId := new(big.Int).SetBytes(log.Topics[1][:])
					event := struct {
						Amount   *big.Int
						Duration *big.Int
					}{}
					err := systemstaking.ABI.UnpackIntoInterface(&event, "BucketExpanded", log.Data)
					if err != nil {
						return errors.WithStack(err)
					}
					oldDecmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, bucketId)
					}
					decmailAmount := decimal.NewFromBigInt(event.Amount, 0)
					info, err := b.getBucketInfoAddressByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					duration := time.Duration(event.Duration.Uint64()) * time.Second
					autoStake := true
					stakeAmount := decmailAmount
					voteWeight := systemstaking.GetVoteWeight(blk.Height(), duration, stakeAmount.BigInt(), autoStake, false)

					stakingBucket = models.SystemStakingBucketV3Record{
						SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
							BlockHeight:          blk.Height(),
							BucketID:             bucketId.Uint64(),
							Sender:               sender.String(),
							OwnerAddress:         info.OwnerAddress,
							ActHash:              actHash,
							CreateTime:           info.CreateTime,
							StakeStartTime:       info.StakeStartTime,
							UnstakeStartTime:     info.UnstakeStartTime,
							StakedAmount:         stakeAmount,
							VotingPower:          decimal.NewFromBigInt(voteWeight, 0),
							DelegateOwnerAddress: info.DelegateOwnerAddress,
							Amount:               decmailAmount.Sub(oldDecmailAmount),
							EventType:            "BucketExpanded",
							AutoStake:            autoStake,
							Duration:             uint32(duration.Seconds()),
							DurationType:         info.DurationType,
							Timestamp:            blk.Timestamp().Unix(),
							Final:                true,
						},
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "DelegateChanged": // DelegateChanged(uint256 indexed bucketId, address newDelegate)
					bucketId := new(big.Int).SetBytes(log.Topics[1][:])
					event := struct {
						NewDelegate common.Address
					}{}
					err := systemstaking.ABI.UnpackIntoInterface(&event, "DelegateChanged", log.Data)
					if err != nil {
						return errors.WithStack(err)
					}
					cadidateAddr, _ := address.FromHex(event.NewDelegate.String())
					info, err := b.getBucketInfoAddressByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					stakeAmount, _ := decimal.NewFromString(info.StakedAmount)
					votingPower, _ := decimal.NewFromString(info.VotingPower)
					stakingBucket = models.SystemStakingBucketV3Record{
						SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
							BlockHeight:          blk.Height(),
							BucketID:             bucketId.Uint64(),
							Sender:               sender.String(),
							OwnerAddress:         info.OwnerAddress,
							ActHash:              actHash,
							CreateTime:           info.CreateTime,
							StakeStartTime:       info.StakeStartTime,
							UnstakeStartTime:     info.UnstakeStartTime,
							StakedAmount:         stakeAmount,
							VotingPower:          votingPower,
							DelegateOwnerAddress: cadidateAddr.String(),
							Amount:               zeroAmount,
							EventType:            "DelegateChanged",
							AutoStake:            info.AutoStake,
							Duration:             info.Duration,
							DurationType:         info.DurationType,
							Timestamp:            blk.Timestamp().Unix(),
							Final:                true,
						},
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "Withdrawal": // Withdrawal(uint256 indexed bucketId, address indexed recipient)
					bucketId := new(big.Int).SetBytes(log.Topics[1][:])
					recipient, _ := address.FromBytes(log.Topics[2][:])

					info, err := b.getBucketInfoAddressByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					stakingBucket = models.SystemStakingBucketV3Record{
						SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
							BlockHeight:          blk.Height(),
							BucketID:             bucketId.Uint64(),
							CreateTime:           info.CreateTime,
							StakeStartTime:       info.StakeStartTime,
							UnstakeStartTime:     info.UnstakeStartTime,
							StakedAmount:         decimal.NewFromInt(0),
							VotingPower:          decimal.NewFromInt(0),
							OwnerAddress:         info.OwnerAddress,
							Sender:               sender.String(),
							Recipient:            recipient.String(),
							ActHash:              actHash,
							DelegateOwnerAddress: info.DelegateOwnerAddress,
							AutoStake:            false,
							EventType:            "Withdrawal",
							Duration:             0,
							DurationType:         info.DurationType,
							Amount:               decimal.NewFromInt(0),
							Timestamp:            blk.Timestamp().Unix(),
							Final:                true,
						},
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "Donated": // Donated(uint256 indexed bucketId, address indexed beneficiary, uint256 amount)
					bucketId := new(big.Int).SetBytes(log.Topics[1][:])
					// beneficiary, _ := address.FromBytes(log.Topics[2][:])
					info, err := b.getBucketInfoAddressByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return err
					}
					event := struct {
						Amount *big.Int
					}{}
					err = systemstaking.ABI.UnpackIntoInterface(&event, "Donated", log.Data)
					if err != nil {
						return errors.WithStack(err)
					}
					decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketId.Uint64())
					if err != nil {
						return err
					}
					stakedAmount := decmailAmount.Sub(decimal.NewFromBigInt(event.Amount, 0))
					voteWeight := systemstaking.GetVoteWeight(blk.Height(), time.Duration(info.Duration)*time.Second, stakedAmount.Coefficient(), info.AutoStake, false)
					stakingBucket = models.SystemStakingBucketV3Record{
						SystemStakingBucketRecordBase: models.SystemStakingBucketRecordBase{
							BlockHeight:          blk.Height(),
							BucketID:             bucketId.Uint64(),
							Sender:               sender.String(),
							OwnerAddress:         info.OwnerAddress,
							ActHash:              actHash,
							CreateTime:           info.CreateTime,
							StakeStartTime:       info.StakeStartTime,
							StakedAmount:         stakedAmount,
							VotingPower:          decimal.NewFromBigInt(voteWeight, 0),
							UnstakeStartTime:     info.UnstakeStartTime,
							DelegateOwnerAddress: info.DelegateOwnerAddress,
							Amount:               decimal.NewFromBigInt(event.Amount, 0).Mul(decimal.NewFromInt(-1)),
							EventType:            "Donated",
							AutoStake:            info.AutoStake,
							Duration:             info.Duration,
							DurationType:         info.DurationType,
							Timestamp:            blk.Timestamp().Unix(),
							Final:                true,
						},
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				}
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return errors.Wrap(err, "")
}

func (b systemStakingBucketPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b systemStakingBucketPlugin) Version() string {
	return VERSION
}

func (b systemStakingBucketPlugin) getBucketInfoAddressByBucketID(tx *gorm.DB, bucketID uint64) (*systemstaking.BucketInfo, error) {
	var bi systemstaking.BucketInfo
	if err := tx.Model(&models.SystemStakingBucketV3Record{}).Select("owner_address,delegate_owner_address,staked_amount,amount,voting_power,auto_stake,duration,duration_type,create_time,stake_start_time,unstake_start_time,muted").Where("bucket_id=?", bucketID).Last(&bi).Error; err != nil {
		return nil, err
	}
	return &bi, nil
}

func (b systemStakingBucketPlugin) getBucketSumAmountByBucketID(tx *gorm.DB, bucketID uint64) (decimal.Decimal, error) {
	var amount sql.NullString
	zero := decimal.NewFromInt(0)
	if err := tx.Model(&models.SystemStakingBucketV3Record{}).Select("sum(amount)").Where("bucket_id=?", bucketID).Scan(&amount).Error; err != nil {
		return zero, err
	}
	if amount.String == "" {
		return zero, nil
	}
	decmailAmount, err := decimal.NewFromString(amount.String)
	if err != nil {
		return zero, err
	}
	return decmailAmount, nil
}

// exported
var Plugin = systemStakingBucketPlugin{}
