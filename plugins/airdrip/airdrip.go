package main

import (
	"context"
	"log"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.1.0"

var successStatus = uint64(1)

var REGISTRATION, CLAIM, EXCHANGE, REDEMPTION, ADD_ASSET, DRIP, TERM hash.Hash256

func init() {
	var err error
	REGISTRATION, err = hash.HexStringToHash256("0xf6c60b059f60c86b2d612b237ac38bd87bc1f68f87600cd360d68351af1ca95f")
	if err != nil {
		log.Fatal(err)
	}
	CLAIM, err = hash.HexStringToHash256("0x47cee97cb7acd717b3c0aa1435d004cd5b3c8c57d70dbceb4e4458bbd60e39d4")
	if err != nil {
		log.Fatal(err)
	}
	EXCHANGE, err = hash.HexStringToHash256("0x26981b9aefbb0f732b0264bd34c255e831001eb50b06bc85b32cc39e14389721")
	if err != nil {
		log.Fatal(err)
	}
	REDEMPTION, err = hash.HexStringToHash256("0xa28d80c9910787c0c058ed9b50c577f1389264bf61563fa45529e0771976f562")
	if err != nil {
		log.Fatal(err)
	}
	ADD_ASSET, err = hash.HexStringToHash256("0x0f0ce87d28aac0c07b026cb6025932cb4009646aeac7965d4cf463fdb1dd9ce0")
	if err != nil {
		log.Fatal(err)
	}
	DRIP, err = hash.HexStringToHash256("0xf616b63fa85386183d388f90cb6006662c309731f969461cd877ee055b306e3a")
	if err != nil {
		log.Fatal(err)
	}
	TERM, err = hash.HexStringToHash256("0x98f0d489a1bb3ac90294374182d5b02db885776732c344172c51f27802be6531")
	if err != nil {
		log.Fatal(err)
	}
}

type airdripPlugin struct {
	accountant string
	exchange   string
}

func (b airdripPlugin) Name() string {
	return "airdrip"
}

func (b airdripPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b airdripPlugin) Start(ctx context.Context) error {
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
	b.accountant = cfg.Accountant
	b.exchange = cfg.Exchange
	return nil
}

func (b airdripPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		// TODO (chenzhen): parse topics and data, and fill into tables
		for _, receipt := range blk.Receipts {
			if receipt.Status != successStatus {
				continue
			}
			for _, log := range receipt.Logs() {
				switch log.Address {
				case b.accountant:
					/**
					 * event Claim(address indexed user, uint256 points);
					 * event NewTerm(uint256 indexed term, uint256 termHeight);
					 * event Exchange(address indexed user, uint256 amount, uint256 exchangeRate);
					 * event Registration(address indexed user, uint256 expireAt);
					 */
					switch log.Topics[0] {
					case CLAIM:
					case TERM:
					case EXCHANGE:
					case REGISTRATION:
					}
				case b.exchange:
					/**
					 * event Redemption(address indexed user, address indexed asset, uint256 amount, uint256 points);
					 * event AssetAdded(address indexed provider, address indexed asset, uint256 amount, uint256 endBlock, uint256 konstante);
					 * event AssetDripped(address indexed asset, uint256 volume);
					 */
					switch log.Topics[0] {
					case REDEMPTION:
					case ADD_ASSET:
					case DRIP:
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
