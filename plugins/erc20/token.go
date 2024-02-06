package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	slog "github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ERC20ABI is the ABI of ERC20 contract
/*

erc20 start block height: 864017
WIOTX start block height: 12014723
*/
const VERSION = "2.2.0"

const (
	WIOTXContractAddress = "io15qr5fzpxsnp7garl4m7k355rafzqn8grrm0grz"
	errFailedInsertTable = "failed to insert table data"
)

var (
	erc20ABI   abi.ABI
	Transfer   hash.Hash256
	Approval   hash.Hash256
	Deposit    hash.Hash256
	Withdrawal hash.Hash256
)

func initAddress() error {
	var err error
	//Transfer(address,address,uint256)
	Transfer, err = hash.HexStringToHash256("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	if err != nil {
		return err
	}
	//Approval(address,address,uint256)
	Approval, err = hash.HexStringToHash256("8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")
	if err != nil {
		return err
	}

	//for special case wiotx
	//Deposit(address,uint256)
	Deposit, err = hash.HexStringToHash256(depositSha3)
	if err != nil {
		return err
	}
	//Withdrawal(address,uint256)
	Withdrawal, err = hash.HexStringToHash256(withdrawalSha3)
	if err != nil {
		return err
	}
	erc20ABI, err = abi.JSON(strings.NewReader(ERC20ABI))
	if err != nil {
		return err
	}
	return nil
}

type tokenPlugin struct {
}

func (b tokenPlugin) Name() string {
	return "erc20"
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(),
		&models.Erc20Transfer{},
		&models.Erc20Holder{},
		&models.Erc20Approval{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b tokenPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, receipt := range blk.Receipts {
			if receipt.Status != successStatus {
				continue
			}
			actionHash := hex.EncodeToString(receipt.ActionHash[:])
			for _, log := range receipt.Logs() {
				if log.Address == "" || len(log.Topics) < 2 {
					continue
				}
				data := hex.EncodeToString(log.Data)
				var topics string
				for _, t := range log.Topics {
					topics += hex.EncodeToString(t[:])
				}
				//skip if not erc20 or wiotx
				if !isErc20(log.Address, topics, data) {
					continue
				}
				var holders []string
				switch log.Topics[0] {
				/**
				 * Transfer(address indexed from, address indexed to, uint256 value);
				 */
				case Transfer:
					event := struct {
						From  common.Address
						To    common.Address
						Value *big.Int
					}{}
					err := kernel.UnpackLog(erc20ABI, &event, "Transfer", log)
					if err != nil {
						return err
					}
					amount := decimal.NewFromBigInt(event.Value, 0)
					fromAddr, _ := address.FromBytes(event.From.Bytes())
					toAddr, _ := address.FromBytes(event.To.Bytes())
					model := models.Erc20Transfer{
						BlockHeight:     blk.Height(),
						ActionHash:      actionHash,
						ContractAddress: log.Address,
						Amount:          amount,
						Sender:          fromAddr.String(),
						Recipient:       toAddr.String(),
						Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
					}
					if err := tx.Create(&model).Error; err != nil {
						return errors.Wrap(err, errFailedInsertTable)
					}
					holders = []string{fromAddr.String(), toAddr.String()}

				//Approval(address indexed owner, address indexed spender, uint256 value);
				case Approval:
					event := struct {
						Owner   common.Address
						Spender common.Address
						Value   *big.Int
					}{}
					err := kernel.UnpackLog(erc20ABI, &event, "Approval", log)
					if err != nil {
						return err
					}
					amount := decimal.NewFromBigInt(event.Value, 0)
					owner, _ := address.FromBytes(event.Owner.Bytes())
					spender, _ := address.FromBytes(event.Spender.Bytes())
					model := models.Erc20Approval{
						BlockHeight:     blk.Height(),
						ActionHash:      actionHash,
						ContractAddress: log.Address,
						Amount:          amount,
						Owner:           owner.String(),
						Spender:         spender.String(),
						Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
					}
					if err := tx.Create(&model).Error; err != nil {
						return errors.Wrap(err, errFailedInsertTable)
					}
				//Deposit(address indexed dst, uint wad);
				case Deposit:
					event := struct {
						Dst common.Address
						Wad *big.Int
					}{}
					err := kernel.UnpackLog(erc20ABI, &event, "Deposit", log)
					if err != nil {
						return errors.WithMessagef(err, "Deposit event: %v", &event)
					}
					amount := decimal.NewFromBigInt(event.Wad, 0)
					to, _ := address.FromBytes(event.Dst.Bytes())
					model := models.Erc20Transfer{
						BlockHeight:     blk.Height(),
						ActionHash:      actionHash,
						ContractAddress: log.Address,
						Amount:          amount,
						Sender:          "",
						Recipient:       to.String(),
						Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
					}
					if err := tx.Create(&model).Error; err != nil {
						return errors.Wrap(err, errFailedInsertTable)
					}
					holders = []string{to.String()}
				//Withdrawal(address indexed src, uint wad);
				case Withdrawal:
					event := struct {
						Src common.Address
						Wad *big.Int
					}{}
					err := kernel.UnpackLog(erc20ABI, &event, "Withdrawal", log)
					if err != nil {
						return errors.WithMessagef(err, "Withdrawal event: %v", &event)
					}
					amount := decimal.NewFromBigInt(event.Wad, 0)
					from, _ := address.FromBytes(event.Src.Bytes())
					model := models.Erc20Transfer{
						BlockHeight:     blk.Height(),
						ActionHash:      actionHash,
						ContractAddress: log.Address,
						Amount:          amount,
						Sender:          from.String(),
						Recipient:       "",
						Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
					}
					if err := tx.Create(&model).Error; err != nil {
						return errors.Wrap(err, errFailedInsertTable)
					}
					holders = []string{from.String()}
				default:
					var topics string
					for _, t := range log.Topics {
						topics = topics + hex.EncodeToString(t[:]) + "\t"
					}
					slog.L().Warn("unknown event", zap.String("contract", log.Address), zap.Uint64("blockHeight", log.BlockHeight), zap.String("topics", topics))

				}
				for _, addr := range holders {
					if addr == "" {
						continue
					}
					model := models.Erc20Holder{
						ContractAddress: log.Address,
						Holder:          addr,
					}
					if err := tx.Where("contract_address = ? and holder= ?", log.Address, addr).First(&model).Error; err != nil {
						if err != gorm.ErrRecordNotFound {
							return err
						}
						if err := tx.Create(&model).Error; err != nil {
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

func (b tokenPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b tokenPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = tokenPlugin{}
