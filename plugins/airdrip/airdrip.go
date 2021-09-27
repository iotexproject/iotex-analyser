package main

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.1.0"

var successStatus = uint64(1)

var REGISTRATION, CLAIM, EXCHANGE, REDEMPTION, ADD_ASSET, DRIP, NEW_TERM hash.Hash256
var accountantABI, exchangeABI abi.ABI

func initAddress() error {
	var err error
	accountantABI, err = abi.JSON(strings.NewReader(AccountantABI))
	if err != nil {
		return err
	}
	REGISTRATION, err = hash.HexStringToHash256("f6c60b059f60c86b2d612b237ac38bd87bc1f68f87600cd360d68351af1ca95f")
	if err != nil {
		return err
	}
	CLAIM, err = hash.HexStringToHash256("47cee97cb7acd717b3c0aa1435d004cd5b3c8c57d70dbceb4e4458bbd60e39d4")
	if err != nil {
		return err
	}
	EXCHANGE, err = hash.HexStringToHash256("26981b9aefbb0f732b0264bd34c255e831001eb50b06bc85b32cc39e14389721")
	if err != nil {
		return err
	}
	REDEMPTION, err = hash.HexStringToHash256("a28d80c9910787c0c058ed9b50c577f1389264bf61563fa45529e0771976f562")
	if err != nil {
		return err
	}
	ADD_ASSET, err = hash.HexStringToHash256("0f0ce87d28aac0c07b026cb6025932cb4009646aeac7965d4cf463fdb1dd9ce0")
	if err != nil {
		return err
	}
	DRIP, err = hash.HexStringToHash256("f616b63fa85386183d388f90cb6006662c309731f969461cd877ee055b306e3a")
	if err != nil {
		return err
	}
	NEW_TERM, err = hash.HexStringToHash256("98f0d489a1bb3ac90294374182d5b02db885776732c344172c51f27802be6531")
	if err != nil {
		return err
	}
	return nil
}

type airdripPlugin struct {
}

func (b airdripPlugin) Name() string {
	return "airdrip"
}

func (b airdripPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b airdripPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.DB().AutoMigrate(
		&models.AirdripAddAsset{},
		&models.AirdripClaim{},
		&models.AirdripDrip{},
		&models.AirdripExchange{},
		&models.AirdripRedemption{},
		&models.AirdripRegistration{},
		&models.AirdripTerm{},
	); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	cfg, err := loadAirdripConfig()
	if err != nil {
		return err
	}

	log.S().Debugf("load airdrip config: %+v", cfg)
	return nil
}

func (b airdripPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, receipt := range blk.Receipts {
			if receipt.Status != successStatus {
				continue
			}
			for _, log := range receipt.Logs() {
				topics := log.Topics
				if log.Address == "" {
					continue
				}
				switch log.Address {
				case Default.Accountant:
					/**
					 * event Claim(address indexed user, uint256 points);
					 * event NewTerm(uint256 indexed term, uint256 termHeight);
					 * event Exchange(address indexed user, uint256 amount, uint256 exchangeRate);
					 * event Registration(address indexed user, uint256 expireAt);
					 */
					switch topics[0] {
					case CLAIM:
						event := new(AccountantClaim)
						if err := UnpackLog(accountantABI, event, "Claim", log); err != nil {
							return err
						}
						m := models.AirdripClaim{
							BlockHeight: blk.Height(),
							User:        event.User.String(),
							Amount:      decimal.NewFromBigInt(event.Points, 0),
						}
						if err := tx.Create(&m).Error; err != nil {
							return err
						}
					case NEW_TERM:
						event := new(AccountantNewTerm)
						if err := UnpackLog(accountantABI, event, "NewTerm", log); err != nil {
							return err
						}
						m := models.AirdripTerm{
							BlockHeight: blk.Height(),
							Number:      event.Term.Uint64(),
							Height:      event.TermHeight.Uint64(),
						}
						if err := tx.Create(&m).Error; err != nil {
							return err
						}
					case EXCHANGE:
						event := new(AccountantExchange)
						if err := UnpackLog(accountantABI, event, "Exchange", log); err != nil {
							return err
						}
						m := models.AirdripExchange{
							BlockHeight: blk.Height(),
							User:        event.User.String(),
							Amount:      decimal.NewFromBigInt(event.Amount, 0),
							Rate:        event.ExchangeRate.Uint64(),
						}
						if err := tx.Create(&m).Error; err != nil {
							return err
						}
					case REGISTRATION:
						event := new(AccountantRegistration)
						if err := UnpackLog(accountantABI, event, "Registration", log); err != nil {
							return err
						}
						m := models.AirdripRegistration{
							BlockHeight: blk.Height(),
							User:        event.User.String(),
							ExpireAt:    event.ExpireAt.Uint64(),
						}
						if err := tx.Create(&m).Error; err != nil {
							return err
						}
					}
				case Default.Exchange:
					/**
					 * event Redemption(address indexed user, address indexed asset, uint256 amount, uint256 points);
					 * event AssetAdded(address indexed provider, address indexed asset, uint256 amount, uint256 endBlock, uint256 konstante);
					 * event AssetDripped(address indexed asset, uint256 volume);
					 */
					switch topics[0] {
					case REDEMPTION:
						event := new(ExchangeRedemption)
						if err := UnpackLog(exchangeABI, event, "Redemption", log); err != nil {
							return err
						}
						m := models.AirdripRedemption{
							BlockHeight: blk.Height(),
							User:        event.User.String(),
							Asset:       event.Asset.String(),
							Amount:      decimal.NewFromBigInt(event.Amount, 0),
							Points:      decimal.NewFromBigInt(event.Points, 0),
						}
						if err := tx.Create(&m).Error; err != nil {
							return err
						}
					case ADD_ASSET:
						event := new(ExchangeAssetAdded)
						if err := UnpackLog(exchangeABI, event, "AssetAdded", log); err != nil {
							return err
						}
						m := models.AirdripAddAsset{
							BlockHeight: blk.Height(),
							Provider:    event.Provider.String(),
							Asset:       event.Asset.String(),
							Amount:      decimal.NewFromBigInt(event.Amount, 0),
							EndBlock:    event.EndBlock.Uint64(),
							Konstante:   event.Konstante.Uint64(),
						}
						if err := tx.Create(&m).Error; err != nil {
							return err
						}
					case DRIP:
						event := new(ExchangeAssetDripped)
						if err := UnpackLog(exchangeABI, event, "AssetDripped", log); err != nil {
							return err
						}
						m := models.AirdripDrip{
							BlockHeight: blk.Height(),
							Asset:       event.Asset.String(),
							Volume:      decimal.NewFromBigInt(event.Volume, 0),
						}
						if err := tx.Create(&m).Error; err != nil {
							return err
						}
					}
				}
			}
		}

		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return err
}
func (b airdripPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b airdripPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = airdripPlugin{}
