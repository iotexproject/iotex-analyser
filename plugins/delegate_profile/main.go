package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.0.0"

const (
	ProfileUpdatedSHA3                = "217aa5ef0b78f028d51fd573433bdbe2daf6f8505e6a71f3af1393c8440b341b"
	blockRewardPortion                = "blockRewardPortion"
	epochRewardPortion                = "epochRewardPortion"
	foundationRewardPortion           = "foundationRewardPortion"
	RewardPortionContract             = "io1lfl4ppn2c3wcft04f0rk0jy9lyn4pcjcm7638u"
	RewardportionContractDeployHeight = 5095225
	successStatus                     = uint64(1)
)

var (
	delegateProfileABI abi.ABI
	ProfileUpdated     hash.Hash256
)

func initAddress() error {
	var err error
	//ProfileUpdated(address,string,bytes)
	ProfileUpdated, err = hash.HexStringToHash256(ProfileUpdatedSHA3)
	if err != nil {
		return err
	}
	delegateProfileABI, err = abi.JSON(strings.NewReader(DelegateProfileABI))
	if err != nil {
		return err
	}
	return nil
}

type delegateProfilePlugin struct {
}

func (b delegateProfilePlugin) Name() string {
	return "delegateProfile"
}

func (b delegateProfilePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b delegateProfilePlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(),
		&models.DelegateProfileUpdated{},
	); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b delegateProfilePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if blk.Height() <= RewardportionContractDeployHeight {
			return nil
		}
		for _, receipt := range blk.Receipts {
			if receipt.Status != successStatus {
				continue
			}
			actionHash := hex.EncodeToString(receipt.ActionHash[:])
			for _, log := range receipt.Logs() {
				if log.Address != RewardPortionContract || len(log.Topics) < 1 {
					continue
				}

				switch log.Topics[0] {
				/**
				 * ProfileUpdated(address delegate, string name, bytes value)
				 */
				case ProfileUpdated:
					event := DelegateProfileProfileUpdated{}
					err := kernel.UnpackLog(delegateProfileABI, &event, "ProfileUpdated", log)
					if err != nil {
						return err
					}
					delegate, _ := address.FromBytes(event.Delegate.Bytes())
					name := event.Name
					value := decimal.NewFromBigInt(big.NewInt(0).SetBytes(event.Value), 0)
					model := models.DelegateProfileUpdated{
						BlockHeight: blk.Height(),
						ActionHash:  actionHash,
						Delegate:    delegate.String(),
						Name:        name,
						Value:       value,
					}
					if err := tx.Create(&model).Error; err != nil {
						return errors.Wrap(err, "failed to insert table data")
					}

				}

			}
		}

		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return err
}

func (b delegateProfilePlugin) Stop(ctx context.Context) error {
	return nil
}

func (b delegateProfilePlugin) Version() string {
	return VERSION
}

// exported
var Plugin = delegateProfilePlugin{}
