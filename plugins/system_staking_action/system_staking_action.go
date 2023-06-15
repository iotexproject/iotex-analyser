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

const VERSION = "2.1.2"

const (
	StakingProtocolAddress         = "io1qnpz47hx5q6r3w876axtrn6yz95d70cjl35r53"
	errBucketSumAmount             = "getBucketSumAmountByBucketID error, bucketID: %d"
	errBucketInfoAddressByBucketID = "getBucketInfoAddressByBucketID error"
)

type systemStakingActionPlugin struct {
}

func (b systemStakingActionPlugin) Name() string {
	return "system_staking_action"
}

func (b systemStakingActionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b systemStakingActionPlugin) DependentPlugins() []string {
	return []string{"candidate"}
}

func (b systemStakingActionPlugin) Start(ctx context.Context) error {
	if err := initContract(); err != nil {
		return errors.Wrap(err, "cannot init contract")
	}
	if err := db.AutoMigrate(b.Name(), &models.SystemStakingActions{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return db.UpdateIndexHeight(b.Name(), config.Default.Genesis.Poll.SystemStakingContractHeight)
}

func (b systemStakingActionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {

	err := db.DB().Transaction(func(tx *gorm.DB) error {
		var stakingAction models.SystemStakingActions
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
					event := struct {
						Delegate common.Address
						Amount   *big.Int
						Duration *big.Int
					}{}
					err := _systemStakingContractABI.UnpackIntoInterface(&event, "Staked", log.Data)
					if err != nil {
						return errors.WithStack(err)
					}
					cadidateAddr, _ := address.FromHex(event.Delegate.String())
					stakingAction = models.SystemStakingActions{
						BlockHeight:  blk.Height(),
						TokenID:      tokenID.Uint64(),
						Sender:       sender.String(),
						OwnerAddress: sender.String(),
						ActHash:      actHash,
						Candidate:    cadidateAddr.String(),
						Amount:       decimal.NewFromBigInt(event.Amount, 0),
						EventType:    "Staked",
						AutoStake:    true, //default true
						Duration:     durationDays(uint32(event.Duration.Uint64())),
					}
					if err := tx.Create(&stakingAction).Error; err != nil {
						return err
					}
				case "Transfer":
					from, _ := address.FromBytes(log.Topics[1][:])
					to, _ := address.FromBytes(log.Topics[2][:])
					tokenID := new(big.Int).SetBytes(log.Topics[3][:])
					decmailAmount, err := getTokenSumAmountByTokenID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, tokenID)
					}
					info, err := getTokenInfoAddressByTokenID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					if from.String() == address.ZeroAddress {
						//mint skip
						stakingAction = models.SystemStakingActions{
							BlockHeight:  blk.Height(),
							TokenID:      tokenID.Uint64(),
							OwnerAddress: to.String(),
							Sender:       sender.String(),
							ActHash:      actHash,
							Candidate:    "",
							AutoStake:    false,
							EventType:    "Transfer",
							Duration:     0,
							Amount:       decimal.NewFromInt(0),
						}
					} else {
						stakingAction = models.SystemStakingActions{
							BlockHeight:  blk.Height(),
							TokenID:      tokenID.Uint64(),
							OwnerAddress: from.String(),
							Sender:       sender.String(),
							ActHash:      actHash,
							Candidate:    info.Candidate,
							AutoStake:    info.AutoStake,
							EventType:    "Transfer",
							Duration:     info.Duration,
							Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
						}
						if err := tx.Create(&stakingAction).Error; err != nil {
							return err
						}
						stakingAction = models.SystemStakingActions{
							BlockHeight:  blk.Height(),
							TokenID:      tokenID.Uint64(),
							OwnerAddress: to.String(),
							Sender:       sender.String(),
							ActHash:      actHash,
							Candidate:    info.Candidate,
							AutoStake:    info.AutoStake,
							EventType:    "Transfer",
							Duration:     info.Duration,
							Amount:       decmailAmount,
						}
					}
					if err := tx.Create(&stakingAction).Error; err != nil {
						return err
					}
				case "Unstaked": //Unstaked(uint256 indexed tokenId, address indexed recipient)
					tokenID := new(big.Int).SetBytes(log.Topics[1][:])
					decmailAmount, err := getTokenSumAmountByTokenID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, tokenID)
					}
					info, err := getTokenInfoAddressByTokenID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					stakingAction = models.SystemStakingActions{
						BlockHeight:  blk.Height(),
						TokenID:      tokenID.Uint64(),
						OwnerAddress: info.OwnerAddress,
						Sender:       sender.String(),
						ActHash:      actHash,
						Candidate:    info.Candidate,
						AutoStake:    info.AutoStake,
						EventType:    "Unstaked",
						Duration:     info.Duration,
						Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
					}
					if err := tx.Create(&stakingAction).Error; err != nil {
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
					info, err := getTokenInfoAddressByTokenID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					stakingAction = models.SystemStakingActions{
						BlockHeight:  blk.Height(),
						TokenID:      tokenID.Uint64(),
						Sender:       sender.String(),
						OwnerAddress: info.OwnerAddress,
						ActHash:      actHash,
						Candidate:    info.Candidate,
						Amount:       zeroAmount,
						EventType:    "Locked",
						AutoStake:    true, //locked must auto stake
						Duration:     durationDays(uint32(event.Duration.Uint64())),
					}
					if err := tx.Create(&stakingAction).Error; err != nil {
						return err
					}
				case "Unlocked":
					tokenID := new(big.Int).SetBytes(log.Topics[1][:])
					info, err := getTokenInfoAddressByTokenID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					stakingAction = models.SystemStakingActions{
						BlockHeight:  blk.Height(),
						TokenID:      tokenID.Uint64(),
						Sender:       sender.String(),
						OwnerAddress: info.OwnerAddress,
						ActHash:      actHash,
						Candidate:    info.Candidate,
						Amount:       zeroAmount,
						EventType:    "Unlocked",
						AutoStake:    false, //unlocked means auto stake is false
						Duration:     info.Duration,
					}
					if err := tx.Create(&stakingAction).Error; err != nil {
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
						tokenAmount, err := getTokenSumAmountByTokenID(tx, tokenID.Uint64())
						if err != nil {
							return errors.Wrapf(err, errBucketSumAmount, tokenID)
						}
						info, err := getTokenInfoAddressByTokenID(tx, tokenID.Uint64())
						if err != nil {
							return errors.Wrap(err, errBucketInfoAddressByBucketID)
						}
						stakingAction = models.SystemStakingActions{
							BlockHeight:  blk.Height(),
							TokenID:      tokenID.Uint64(),
							Sender:       sender.String(),
							OwnerAddress: info.OwnerAddress,
							ActHash:      actHash,
							Candidate:    info.Candidate,
							Amount:       tokenAmount.Mul(decimal.NewFromInt(-1)),
							EventType:    "Merged",
							AutoStake:    info.AutoStake,
							Duration:     info.Duration,
						}
						if err := tx.Create(&stakingAction).Error; err != nil {
							return err
						}
					}
					tokenID := event.TokenIds[0]
					tokenAmount, err := getTokenSumAmountByTokenID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, tokenID)
					}
					info, err := getTokenInfoAddressByTokenID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					stakingAction = models.SystemStakingActions{
						BlockHeight:  blk.Height(),
						TokenID:      tokenID.Uint64(),
						Sender:       sender.String(),
						OwnerAddress: info.OwnerAddress,
						ActHash:      actHash,
						Candidate:    info.Candidate,
						Amount:       decmailAmount.Sub(tokenAmount),
						EventType:    "Merged",
						AutoStake:    true,
						Duration:     durationDays(uint32(event.Duration.Uint64())),
					}
					if err := tx.Create(&stakingAction).Error; err != nil {
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
					oldDecmailAmount, err := getTokenSumAmountByTokenID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, tokenID)
					}
					decmailAmount := decimal.NewFromBigInt(event.Amount, 0)
					info, err := getTokenInfoAddressByTokenID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					stakingAction = models.SystemStakingActions{
						BlockHeight:  blk.Height(),
						TokenID:      tokenID.Uint64(),
						OwnerAddress: info.OwnerAddress,
						Sender:       sender.String(),
						ActHash:      actHash,
						Candidate:    info.Candidate,
						AutoStake:    info.AutoStake,
						EventType:    "BucketExpanded",
						Duration:     info.Duration,
						Amount:       oldDecmailAmount.Mul(decimal.NewFromInt(-1)),
					}
					if err := tx.Create(&stakingAction).Error; err != nil {
						return err
					}
					stakingAction = models.SystemStakingActions{
						BlockHeight:  blk.Height(),
						TokenID:      tokenID.Uint64(),
						Sender:       sender.String(),
						OwnerAddress: info.OwnerAddress,
						ActHash:      actHash,
						Candidate:    info.Candidate,
						Amount:       decmailAmount,
						EventType:    "BucketExpanded",
						AutoStake:    true,
						Duration:     durationDays(uint32(event.Duration.Uint64())),
					}
					if err := tx.Create(&stakingAction).Error; err != nil {
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
					decmailAmount, err := getTokenSumAmountByTokenID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrapf(err, errBucketSumAmount, tokenID)
					}
					cadidateAddr, _ := address.FromHex(event.NewDelegate.String())
					info, err := getTokenInfoAddressByTokenID(tx, tokenID.Uint64())
					if err != nil {
						return errors.Wrap(err, errBucketInfoAddressByBucketID)
					}
					stakingAction = models.SystemStakingActions{
						BlockHeight:  blk.Height(),
						TokenID:      tokenID.Uint64(),
						Sender:       sender.String(),
						OwnerAddress: info.OwnerAddress,
						ActHash:      actHash,
						Candidate:    cadidateAddr.String(),
						Amount:       decmailAmount,
						EventType:    "DelegateChanged",
						AutoStake:    info.AutoStake,
						Duration:     info.Duration,
					}
					if err := tx.Create(&stakingAction).Error; err != nil {
						return err
					}
				}
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return errors.Wrap(err, "")
}

func (b systemStakingActionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b systemStakingActionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = systemStakingActionPlugin{}
