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
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const VERSION = "2.2.3"

const (
	// TransferString is sha3 of xrc20's transfer event,keccak('Transfer(address,address,uint256)')
	TransferString = "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	//Approve(address,address,uint256)
	ApproveString = "8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"

	//ApprovalForAll(address,address,bool)
	ApprovalForAllString = "17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31"

	successStatus = uint64(1)
)

var (
	erc721ABI         abi.ABI
	Transfer          hash.Hash256
	Approval          hash.Hash256
	ApprovalForAll    hash.Hash256
	erc721Contract    = make(map[string]struct{})
	nonErc721Contract = make(map[string]struct{})
)
var (
	errFailedInsertTable = "failed to insert table data"
)

func initAddress() error {
	var err error
	//Transfer(address,address,uint256)
	Transfer, err = hash.HexStringToHash256(TransferString)
	if err != nil {
		return err
	}
	//Approval(address,address,uint256)
	Approval, err = hash.HexStringToHash256(ApproveString)
	if err != nil {
		return err
	}
	//ApprovalForAll(address,address,bool)
	ApprovalForAll, err = hash.HexStringToHash256(ApprovalForAllString)
	if err != nil {
		return err
	}
	erc721ABI, err = abi.JSON(strings.NewReader(ERC721ABI))
	if err != nil {
		return err
	}
	return nil
}

type tokenPlugin struct {
	tipHeight       uint64
	transfers       []*Erc721Transfer
	approvals       []*Erc721Approval
	approvalForAlls []*Erc721ApprovalForAll
	holders         map[string]*Erc721Holder
}

func (b *tokenPlugin) Name() string {
	return "erc721_" + VERSION
}

func (b *tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *tokenPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(),
		&Erc721Transfer{},
		&Erc721Holder{},
		&Erc721Approval{},
		&Erc721ApprovalForAll{},
	); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b *tokenPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	if b.holders == nil {
		b.holders = make(map[string]*Erc721Holder)
	}
	for _, receipt := range blk.Receipts {
		if receipt.Status != successStatus {
			continue
		}
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		for _, log := range receipt.Logs() {
			if _, ok := nonErc721Contract[log.Address]; ok {
				continue
			}
			if _, ok := erc721Contract[log.Address]; !ok {
				ok, err := kernel.IsErc721(log.Address)
				if err != nil {
					return errors.Wrap(err, "failed to check erc721")
				}
				if !ok {
					nonErc721Contract[log.Address] = struct{}{}
					continue
				}
				erc721Contract[log.Address] = struct{}{}
			}
			var holderAddrs []string
			switch log.Topics[0] {
			/**
			 * Transfer(address indexed from, address indexed to, uint256 indexed tokenId);
			 */
			case Transfer:
				event := struct {
					From    common.Address
					To      common.Address
					TokenId *big.Int
				}{}
				err := kernel.UnpackLog(erc721ABI, &event, "Transfer", log)
				if err != nil {
					return errors.WithMessagef(err, "failed to unpack Transfer on action: %s", actionHash)
				}
				tokenID := decimal.NewFromBigInt(event.TokenId, 0)
				fromAddr, _ := address.FromBytes(event.From.Bytes())
				toAddr, _ := address.FromBytes(event.To.Bytes())
				b.transfers = append(b.transfers, &Erc721Transfer{
					BlockHeight:     blk.Height(),
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					TokenId:         tokenID,
					Sender:          fromAddr.String(),
					Recipient:       toAddr.String(),
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				})
				holderAddrs = []string{fromAddr.String(), toAddr.String()}

			//Approval(address indexed owner, address indexed approved, uint256 indexed tokenId);
			case Approval:
				event := struct {
					Owner    common.Address
					Approved common.Address
					TokenId  *big.Int
				}{}
				err := kernel.UnpackLog(erc721ABI, &event, "Approval", log)
				if err != nil {
					return errors.WithMessagef(err, "failed to unpack Approval on action: %s", actionHash)
				}
				tokenID := decimal.NewFromBigInt(event.TokenId, 0)
				owner, _ := address.FromBytes(event.Owner.Bytes())
				approved, _ := address.FromBytes(event.Approved.Bytes())
				b.approvals = append(b.approvals, &Erc721Approval{
					BlockHeight:     blk.Height(),
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					TokenId:         tokenID,
					Owner:           owner.String(),
					Approved:        approved.String(),
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				})
			//ApprovalForAll(address indexed owner, address indexed operator, bool approved);
			case ApprovalForAll:
				event := struct {
					Owner    common.Address
					Operator common.Address
					Approved bool
				}{}
				err := kernel.UnpackLog(erc721ABI, &event, "ApprovalForAll", log)
				if err != nil {
					return errors.WithMessagef(err, "ApprovalForAll event: %v", &event)
				}
				owner, _ := address.FromBytes(event.Owner.Bytes())
				operator, _ := address.FromBytes(event.Operator.Bytes())
				b.approvalForAlls = append(b.approvalForAlls, &Erc721ApprovalForAll{
					BlockHeight:     blk.Height(),
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					Owner:           owner.String(),
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
			for _, addr := range holderAddrs {
				if addr == "" {
					continue
				}
				key := log.Address + ":" + addr
				if _, exists := b.holders[key]; !exists {
					b.holders[key] = &Erc721Holder{
						ContractAddress: log.Address,
						Holder:          addr,
					}
				}
			}
		}
	}
	return nil
}

func (b *tokenPlugin) commit() error {
	transfers := b.transfers
	approvals := b.approvals
	approvalForAlls := b.approvalForAlls
	holders := make([]*Erc721Holder, 0, len(b.holders))
	for _, h := range b.holders {
		holders = append(holders, h)
	}
	b.transfers = nil
	b.approvals = nil
	b.approvalForAlls = nil
	b.holders = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(transfers) > 0 {
			if err := tx.CreateInBatches(transfers, 200).Error; err != nil {
				return errors.Wrap(err, errFailedInsertTable)
			}
		}
		if len(approvals) > 0 {
			if err := tx.CreateInBatches(approvals, 200).Error; err != nil {
				return errors.Wrap(err, errFailedInsertTable)
			}
		}
		if len(approvalForAlls) > 0 {
			if err := tx.CreateInBatches(approvalForAlls, 200).Error; err != nil {
				return errors.Wrap(err, errFailedInsertTable)
			}
		}
		if len(holders) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(holders, 200).Error; err != nil {
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
var Plugin = tokenPlugin{}
