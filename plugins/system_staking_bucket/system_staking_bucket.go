package main

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.2.4"

const (
	errBucketSumAmount             = "getBucketSumAmountByBucketID error, bucketID: %d"
	errBucketInfoAddressByBucketID = "getBucketInfoAddressByBucketID error"
)

type systemStakingBucketPlugin struct {
}

func (b systemStakingBucketPlugin) Name() string {
	return "system_staking_bucket"
}

func (b systemStakingBucketPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b systemStakingBucketPlugin) Start(ctx context.Context) error {
	if err := initContract(); err != nil {
		return errors.Wrap(err, "cannot init contract")
	}
	if err := db.AutoMigrate(b.Name(), &models.SystemStakingBucket{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	height, err := db.GetIndexHeight(b.Name())
	if err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	if height == 0 {
		return db.UpdateIndexHeight(b.Name(), config.Default.Genesis.Poll.SystemStakingContractHeight)
	}
	return nil
}

func (b systemStakingBucketPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		var stakingBucket models.SystemStakingBucket
		actions := make(map[hash.Hash256]action.SealedEnvelope, len(blk.Actions))
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
				if log.Address != config.Default.Genesis.Poll.SystemStakingContractAddress {
					continue
				}
				abiEvent, err := _systemStakingContractABI.EventByID(common.Hash(log.Topics[0]))
				if err != nil {
					return errors.WithStack(err)
				}
				switch abiEvent.Name {
				//event Staked(uint256 indexed tokenId, address delegate, uint256 amount, uint256 duration);
				case "Staked":
					tokenID := new(big.Int).SetBytes(log.Topics[1][:])
					info, err := getBucketInfoAddressByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return err
					}
					event := struct {
						Delegate common.Address
						Amount   *big.Int
						Duration *big.Int
					}{}
					err = _systemStakingContractABI.UnpackIntoInterface(&event, "Staked", log.Data)
					if err != nil {
						return errors.WithStack(err)
					}
					decmailAmount, err := getBucketSumAmountByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return err
					}
					stakedAmount := decmailAmount.Add(decimal.NewFromBigInt(event.Amount, 0))
					duration := durationDays(uint32(event.Duration.Uint64()))
					voteWeight := getVoteWeight(blk.Height(), uint32(duration), stakedAmount.Coefficient(), true, false)
					cadidateAddr, _ := address.FromHex(event.Delegate.String())
					stakingBucket = models.SystemStakingBucket{
						BlockHeight:          blk.Height(),
						BucketID:             tokenID.Uint64(),
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
						Duration:             duration,
						Timestamp:            blk.Timestamp().Unix(),
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "Transfer":
					from, _ := address.FromBytes(log.Topics[1][:])
					to, _ := address.FromBytes(log.Topics[2][:])
					tokenID := new(big.Int).SetBytes(log.Topics[3][:])
					if from.String() == address.ZeroAddress {
						//mint skip
						stakingBucket = models.SystemStakingBucket{
							BlockHeight:          blk.Height(),
							BucketID:             tokenID.Uint64(),
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
							Amount:               decimal.NewFromInt(0),
							Timestamp:            blk.Timestamp().Unix(),
						}
					} else {
						info, err := getBucketInfoAddressByBucketID(tx, tokenID.Uint64())
						if err != nil {
							return errors.Wrap(err, errBucketInfoAddressByBucketID)
						}
						decmailAmount, err := getBucketSumAmountByBucketID(tx, tokenID.Uint64())
						if err != nil {
							return errors.Wrapf(err, errBucketSumAmount, tokenID)
						}
						stakeAmount, _ := decimal.NewFromString(info.StakedAmount)
						votingPower, _ := decimal.NewFromString(info.VotingPower)
						stakingBucket = models.SystemStakingBucket{
							BlockHeight:          blk.Height(),
							BucketID:             tokenID.Uint64(),
							OwnerAddress:         from.String(),
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
							Amount:               decmailAmount.Mul(decimal.NewFromInt(-1)),
							Timestamp:            blk.Timestamp().Unix(),
						}
						if err := tx.Create(&stakingBucket).Error; err != nil {
							return err
						}
						stakingBucket = models.SystemStakingBucket{
							BlockHeight:          blk.Height(),
							BucketID:             tokenID.Uint64(),
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
							Amount:               decmailAmount,
							Timestamp:            blk.Timestamp().Unix(),
						}
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "Unstaked": //Unstaked(uint256 indexed tokenId, address indexed recipient)
					tokenID := new(big.Int).SetBytes(log.Topics[1][:])
					decmailAmount, err := getBucketSumAmountByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, tokenID)
					}
					info, err := getBucketInfoAddressByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}

					stakingBucket = models.SystemStakingBucket{
						BlockHeight:          blk.Height(),
						BucketID:             tokenID.Uint64(),
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
						Amount:               decmailAmount.Mul(decimal.NewFromInt(-1)),
						Timestamp:            blk.Timestamp().Unix(),
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "Locked":
					tokenID := new(big.Int).SetBytes(log.Topics[1][:])
					event := struct {
						Duration *big.Int
					}{}
					err := _systemStakingContractABI.UnpackIntoInterface(&event, "Locked", log.Data)
					if err != nil {
						return errors.WithStack(err)
					}
					info, err := getBucketInfoAddressByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					decmailAmount, err := getBucketSumAmountByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, tokenID)
					}
					duration := durationDays(uint32(event.Duration.Uint64()))
					autoStake := true //locked must auto stake
					voteWeight := getVoteWeight(blk.Height(), uint32(duration), decmailAmount.BigInt(), autoStake, false)
					stakingBucket = models.SystemStakingBucket{
						BlockHeight:          blk.Height(),
						BucketID:             tokenID.Uint64(),
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
						Duration:             duration,
						Timestamp:            blk.Timestamp().Unix(),
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "Unlocked":
					tokenID := new(big.Int).SetBytes(log.Topics[1][:])
					info, err := getBucketInfoAddressByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					decmailAmount, err := getBucketSumAmountByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, tokenID)
					}
					duration := info.Duration
					autoStake := false //unlocked means auto stake is false
					voteWeight := getVoteWeight(blk.Height(), uint32(duration), decmailAmount.BigInt(), autoStake, false)
					stakingBucket = models.SystemStakingBucket{
						BlockHeight:          blk.Height(),
						BucketID:             tokenID.Uint64(),
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
						Timestamp:            blk.Timestamp().Unix(),
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "Merged": //Merged(uint256[] tokenIds, uint256 amount, uint256 duration)
					event := struct {
						TokenIds []*big.Int
						Amount   *big.Int
						Duration *big.Int
					}{}
					err := _systemStakingContractABI.UnpackIntoInterface(&event, "Merged", log.Data)
					if err != nil {
						return errors.WithStack(err)
					}
					decmailAmount := decimal.NewFromBigInt(event.Amount, 0)
					//Here the amount of tokenIDs from the second tokenID are added to the first tokenID
					for _, tokenID := range event.TokenIds[1:] {
						tokenAmount, err := getBucketSumAmountByBucketID(tx, tokenID.Uint64())
						if err != nil {
							return errors.Wrapf(err, errBucketSumAmount, tokenID)
						}
						info, err := getBucketInfoAddressByBucketID(tx, tokenID.Uint64())
						if err != nil {
							return errors.Wrap(err, errBucketInfoAddressByBucketID)
						}
						stakingBucket = models.SystemStakingBucket{
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
							Timestamp:            blk.Timestamp().Unix(),
						}
						if err := tx.Create(&stakingBucket).Error; err != nil {
							return err
						}
					}
					tokenID := event.TokenIds[0]
					tokenAmount, err := getBucketSumAmountByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, tokenID)
					}
					info, err := getBucketInfoAddressByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					duration := durationDays(uint32(event.Duration.Uint64()))
					autoStake := true
					stakeAmount := decmailAmount
					voteWeight := getVoteWeight(blk.Height(), uint32(duration), stakeAmount.BigInt(), autoStake, false)
					stakingBucket = models.SystemStakingBucket{
						BlockHeight:          blk.Height(),
						BucketID:             tokenID.Uint64(),
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
						Duration:             duration,
						Timestamp:            blk.Timestamp().Unix(),
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "BucketExpanded": //BucketExpanded(uint256 indexed tokenId, uint256 amount, uint256 duration);
					tokenID := new(big.Int).SetBytes(log.Topics[1][:])
					event := struct {
						Amount   *big.Int
						Duration *big.Int
					}{}
					err := _systemStakingContractABI.UnpackIntoInterface(&event, "BucketExpanded", log.Data)
					if err != nil {
						return errors.WithStack(err)
					}
					oldDecmailAmount, err := getBucketSumAmountByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, tokenID)
					}
					decmailAmount := decimal.NewFromBigInt(event.Amount, 0)
					info, err := getBucketInfoAddressByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					duration := durationDays(uint32(event.Duration.Uint64()))
					autoStake := true
					stakeAmount := decmailAmount
					oldVoteWeight := getVoteWeight(blk.Height(), info.Duration, oldDecmailAmount.BigInt(), info.AutoStake, false)
					voteWeight := getVoteWeight(blk.Height(), uint32(duration), stakeAmount.BigInt(), autoStake, false)

					stakingBucket = models.SystemStakingBucket{
						BlockHeight:          blk.Height(),
						BucketID:             tokenID.Uint64(),
						OwnerAddress:         info.OwnerAddress,
						Sender:               sender.String(),
						ActHash:              actHash,
						CreateTime:           info.CreateTime,
						StakeStartTime:       info.StakeStartTime,
						UnstakeStartTime:     info.UnstakeStartTime,
						StakedAmount:         oldDecmailAmount,
						VotingPower:          decimal.NewFromBigInt(oldVoteWeight, 0),
						DelegateOwnerAddress: info.DelegateOwnerAddress,
						AutoStake:            info.AutoStake,
						EventType:            "BucketExpanded",
						Duration:             info.Duration,
						Amount:               oldDecmailAmount.Mul(decimal.NewFromInt(-1)),
						Timestamp:            blk.Timestamp().Unix(),
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
					stakingBucket = models.SystemStakingBucket{
						BlockHeight:          blk.Height(),
						BucketID:             tokenID.Uint64(),
						Sender:               sender.String(),
						OwnerAddress:         info.OwnerAddress,
						ActHash:              actHash,
						CreateTime:           info.CreateTime,
						StakeStartTime:       info.StakeStartTime,
						UnstakeStartTime:     info.UnstakeStartTime,
						StakedAmount:         stakeAmount,
						VotingPower:          decimal.NewFromBigInt(voteWeight, 0),
						DelegateOwnerAddress: info.DelegateOwnerAddress,
						Amount:               decmailAmount,
						EventType:            "BucketExpanded",
						AutoStake:            autoStake,
						Duration:             duration,
						Timestamp:            blk.Timestamp().Unix(),
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "DelegateChanged": //DelegateChanged(uint256 indexed tokenId, address newDelegate);
					tokenID := new(big.Int).SetBytes(log.Topics[1][:])
					event := struct {
						NewDelegate common.Address
					}{}
					err := _systemStakingContractABI.UnpackIntoInterface(&event, "DelegateChanged", log.Data)
					if err != nil {
						return errors.WithStack(err)
					}
					cadidateAddr, _ := address.FromHex(event.NewDelegate.String())
					info, err := getBucketInfoAddressByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					stakeAmount, _ := decimal.NewFromString(info.StakedAmount)
					votingPower, _ := decimal.NewFromString(info.VotingPower)
					stakingBucket = models.SystemStakingBucket{
						BlockHeight:          blk.Height(),
						BucketID:             tokenID.Uint64(),
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
						Timestamp:            blk.Timestamp().Unix(),
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
				case "Withdrawal": //Withdrawal(uint256 indexed tokenId, address indexed recipient);
					tokenID := new(big.Int).SetBytes(log.Topics[1][:])
					recipient, _ := address.FromBytes(log.Topics[2][:])

					info, err := getBucketInfoAddressByBucketID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					stakingBucket = models.SystemStakingBucket{
						BlockHeight:          blk.Height(),
						BucketID:             tokenID.Uint64(),
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
						Amount:               decimal.NewFromInt(0),
						Timestamp:            blk.Timestamp().Unix(),
					}
					if err := tx.Create(&stakingBucket).Error; err != nil {
						return err
					}
					// case "OwnershipTransferred": //OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
					// event := struct {
					// 	PreviousOwner common.Address
					// 	NewOwner      common.Address
					// }{}
					// err := _systemStakingContractABI.UnpackIntoInterface(&event, "OwnershipTransferred", log.Data)
					// if err != nil {
					// 	return errors.WithStack(err)
					// }
					// previousOwner, _ := address.FromHex(event.PreviousOwner.String())
					// newOwner, _ := address.FromHex(event.NewOwner.String())

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

// exported
var Plugin = systemStakingBucketPlugin{}
