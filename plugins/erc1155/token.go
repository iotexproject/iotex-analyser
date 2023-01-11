package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
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

const VERSION = "2.2.0"

var (
	errFailedInsertTable = "failed to insert table data"
)
var (
	erc1155ABI         abi.ABI
	HashTransferBatch  hash.Hash256
	HashTransferSingle hash.Hash256
	HashURI            hash.Hash256
	HashApprovalForAll hash.Hash256
)

func initAddress() error {
	var err error
	HashTransferBatch, err = hash.HexStringToHash256(TransferBatchString)
	if err != nil {
		return err
	}
	HashTransferSingle, err = hash.HexStringToHash256(TransferSingleString)
	if err != nil {
		return err
	}
	HashURI, err = hash.HexStringToHash256(URIString)
	if err != nil {
		return err
	}
	HashApprovalForAll, err = hash.HexStringToHash256(ApprovalForAllString)
	if err != nil {
		return err
	}

	erc1155ABI, err = abi.JSON(strings.NewReader(ERC1155ABI))
	if err != nil {
		return err
	}
	return nil
}

type tokenPlugin struct {
}

func (b tokenPlugin) Name() string {
	return "erc1155"
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(),
		&models.Erc1155TransferBatch{},
		&models.Erc1155TransferSingle{},
		&models.Erc1155URI{},
		&models.Erc1155ApprovalForAll{}); err != nil {
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
				if !isErc1155(log.Address, topics, data) {
					continue
				}
				switch log.Topics[0] {
				//TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
				case HashTransferBatch:
					event := struct {
						Operator common.Address
						From     common.Address
						To       common.Address
						Ids      []*big.Int
						Values   []*big.Int
					}{}
					err := kernel.UnpackLog(erc1155ABI, &event, "TransferBatch", log)
					if err != nil {
						return err
					}
					operatorAddr, _ := address.FromBytes(event.Operator.Bytes())
					fromAddr, _ := address.FromBytes(event.From.Bytes())
					toAddr, _ := address.FromBytes(event.To.Bytes())
					ids := make([]string, 0)
					values := make([]string, 0)
					for _, id := range event.Ids {
						ids = append(ids, id.String())
					}
					for _, value := range event.Values {
						values = append(values, value.String())
					}
					jsonIds, _ := json.Marshal(ids)
					jsonValues, _ := json.Marshal(values)
					model := models.Erc1155TransferBatch{
						BlockHeight:     blk.Height(),
						ActionHash:      actionHash,
						ContractAddress: log.Address,
						Operator:        operatorAddr.String(),
						Sender:          fromAddr.String(),
						Recipient:       toAddr.String(),
						IDs:             jsonIds,
						Values:          jsonValues,
						Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
					}
					if err := tx.Create(&model).Error; err != nil {
						return errors.Wrap(err, errFailedInsertTable)
					}

				//TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
				case HashTransferSingle:
					event := struct {
						Operator common.Address
						From     common.Address
						To       common.Address
						Id       *big.Int
						Value    *big.Int
					}{}
					err := kernel.UnpackLog(erc1155ABI, &event, "TransferSingle", log)
					if err != nil {
						return err
					}
					sid := decimal.NewFromBigInt(event.Id, 0)
					value := decimal.NewFromBigInt(event.Value, 0)
					operator, _ := address.FromBytes(event.Operator.Bytes())
					from, _ := address.FromBytes(event.From.Bytes())
					to, _ := address.FromBytes(event.To.Bytes())
					model := models.Erc1155TransferSingle{
						BlockHeight:     blk.Height(),
						ActionHash:      actionHash,
						ContractAddress: log.Address,
						Operator:        operator.String(),
						Sender:          from.String(),
						Recipient:       to.String(),
						SID:             sid,
						Value:           value,
						Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
					}
					if err := tx.Create(&model).Error; err != nil {
						return errors.Wrap(err, errFailedInsertTable)
					}
				//URI(string value, uint256 indexed id)
				case HashURI:
					event := struct {
						Value string
						Id    *big.Int
					}{}
					err := kernel.UnpackLog(erc1155ABI, &event, "URI", log)
					if err != nil {
						return errors.WithMessagef(err, "URI event: %v", &event)
					}
					id := decimal.NewFromBigInt(event.Id, 0)
					model := models.Erc1155URI{
						BlockHeight:     blk.Height(),
						ActionHash:      actionHash,
						ContractAddress: log.Address,
						Value:           event.Value,
						SID:             id,
						Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
					}
					if err := tx.Create(&model).Error; err != nil {
						return errors.Wrap(err, errFailedInsertTable)
					}
				//ApprovalForAll(address indexed account, address indexed operator, bool approved)
				case HashApprovalForAll:
					event := struct {
						Account  common.Address
						Operator common.Address
						Approved bool
					}{}
					err := kernel.UnpackLog(erc1155ABI, &event, "ApprovalForAll", log)
					if err != nil {
						return errors.WithMessagef(err, "Withdrawal event: %v", &event)
					}
					account, _ := address.FromBytes(event.Account.Bytes())
					operator, _ := address.FromBytes(event.Operator.Bytes())
					model := models.Erc1155ApprovalForAll{
						BlockHeight:     blk.Height(),
						ActionHash:      actionHash,
						ContractAddress: log.Address,
						Account:         account.String(),
						Operator:        operator.String(),
						Approved:        event.Approved,
						Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
					}
					if err := tx.Create(&model).Error; err != nil {
						return errors.Wrap(err, errFailedInsertTable)
					}
				default:
					var topics string
					for _, t := range log.Topics {
						topics = topics + hex.EncodeToString(t[:]) + "\t"
					}
					slog.L().Warn("unknown event", zap.String("contract", log.Address), zap.Uint64("blockHeight", log.BlockHeight), zap.String("topics", topics))

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
