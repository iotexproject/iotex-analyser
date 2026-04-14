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
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const VERSION = "2.2.2"

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
	tipHeight       uint64
	transferBatch   []*Erc1155TransferBatch
	transferSingle  []*Erc1155TransferSingle
	uris            []*Erc1155URI
	approvalForAlls []*Erc1155ApprovalForAll
}

func (b *tokenPlugin) Name() string {
	return "erc1155_" + VERSION
}

func (b *tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *tokenPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(),
		&Erc1155TransferBatch{},
		&Erc1155TransferSingle{},
		&Erc1155URI{},
		&Erc1155ApprovalForAll{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b *tokenPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	for _, receipt := range blk.Receipts {
		if receipt.Status != successStatus {
			continue
		}
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		for _, log := range receipt.Logs() {
			if _, ok := nonErc1155Contract[log.Address]; ok {
				continue
			}
			if _, ok := erc1155Contract[log.Address]; !ok {
				ok, err := kernel.IsErc1155(log.Address)
				if err != nil {
					return errors.Wrap(err, "failed to check erc1155")
				}
				if !ok {
					nonErc1155Contract[log.Address] = struct{}{}
					continue
				}
				erc1155Contract[log.Address] = struct{}{}
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
				b.transferBatch = append(b.transferBatch, &Erc1155TransferBatch{
					BlockHeight:     blk.Height(),
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					Operator:        operatorAddr.String(),
					Sender:          fromAddr.String(),
					Recipient:       toAddr.String(),
					IDs:             jsonIds,
					Values:          jsonValues,
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				})

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
				b.transferSingle = append(b.transferSingle, &Erc1155TransferSingle{
					BlockHeight:     blk.Height(),
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					Operator:        operator.String(),
					Sender:          from.String(),
					Recipient:       to.String(),
					SID:             sid,
					Value:           value,
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				})
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
				b.uris = append(b.uris, &Erc1155URI{
					BlockHeight:     blk.Height(),
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					Value:           event.Value,
					SID:             id,
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				})
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
				b.approvalForAlls = append(b.approvalForAlls, &Erc1155ApprovalForAll{
					BlockHeight:     blk.Height(),
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					Account:         account.String(),
					Operator:        operator.String(),
					Approved:        event.Approved,
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				})
			default:
				var topics string
				for _, t := range log.Topics {
					topics = topics + hex.EncodeToString(t[:]) + "\t"
				}
				slog.L().Warn("unknown event", zap.String("contract", log.Address), zap.Uint64("blockHeight", log.BlockHeight), zap.String("topics", topics))
			}
		}
	}
	return nil
}

func (b *tokenPlugin) commit() error {
	transferBatch := b.transferBatch
	transferSingle := b.transferSingle
	uris := b.uris
	approvalForAlls := b.approvalForAlls
	b.transferBatch = nil
	b.transferSingle = nil
	b.uris = nil
	b.approvalForAlls = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(transferBatch) > 0 {
			if err := tx.CreateInBatches(transferBatch, 200).Error; err != nil {
				return errors.Wrap(err, errFailedInsertTable)
			}
		}
		if len(transferSingle) > 0 {
			if err := tx.CreateInBatches(transferSingle, 200).Error; err != nil {
				return errors.Wrap(err, errFailedInsertTable)
			}
		}
		if len(uris) > 0 {
			if err := tx.CreateInBatches(uris, 200).Error; err != nil {
				return errors.Wrap(err, errFailedInsertTable)
			}
		}
		if len(approvalForAlls) > 0 {
			if err := tx.CreateInBatches(approvalForAlls, 200).Error; err != nil {
				return errors.Wrap(err, errFailedInsertTable)
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *tokenPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *tokenPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *tokenPlugin) BatchSize() int {
	return 1000
}

func (b *tokenPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *tokenPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = &tokenPlugin{}
