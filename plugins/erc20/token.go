package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
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
	batchSize     int
	tipHeight     uint64
	erc20Transfer []*Erc20Transfer
	erc20Holder   []*Erc20Holder
	erc20Approval []*Erc20Approval
}

func (b tokenPlugin) Name() string {
	return "erc20"
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) BatchSize() int {
	return b.batchSize
}

func (b *tokenPlugin) Start(ctx context.Context) error {
	var err error
	cfg := &Config{}
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		if err = yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}

	b.batchSize = cfg.BatchSize

	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	return nil
}

func (b tokenPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[0].Height() + uint64(len(blks)) - 1
	return b.commit()
}

func (b tokenPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *tokenPlugin) putBlock(ctx context.Context, blk *block.Block) error {
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
				b.erc20Transfer = append(b.erc20Transfer, &Erc20Transfer{
					BlockHeight:     blk.Height(),
					LogIndex:        log.Index,
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					Amount:          amount.String(),
					Sender:          fromAddr.String(),
					Recipient:       toAddr.String(),
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				})
				b.erc20Holder = append(b.erc20Holder, &Erc20Holder{
					ContractAddress: log.Address,
					Holder:          fromAddr.String(),
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				}, &Erc20Holder{
					ContractAddress: log.Address,
					Holder:          toAddr.String(),
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				})

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
				b.erc20Approval = append(b.erc20Approval, &Erc20Approval{
					BlockHeight:     blk.Height(),
					LogIndex:        log.Index,
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					Amount:          amount.String(),
					Owner:           owner.String(),
					Spender:         spender.String(),
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				})
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
				b.erc20Transfer = append(b.erc20Transfer, &Erc20Transfer{
					BlockHeight:     blk.Height(),
					LogIndex:        log.Index,
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					Amount:          amount.String(),
					Sender:          "",
					Recipient:       to.String(),
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				})
				b.erc20Holder = append(b.erc20Holder, &Erc20Holder{
					ContractAddress: log.Address,
					Holder:          to.String(),
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				})
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
				b.erc20Transfer = append(b.erc20Transfer, &Erc20Transfer{
					BlockHeight:     blk.Height(),
					LogIndex:        log.Index,
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					Amount:          amount.String(),
					Sender:          from.String(),
					Recipient:       "",
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				})
				b.erc20Holder = append(b.erc20Holder, &Erc20Holder{
					ContractAddress: log.Address,
					Holder:          from.String(),
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
	ctx := context.Background()
	if len(b.erc20Transfer) > 0 {
		defer func() {
			b.erc20Transfer = b.erc20Transfer[:0]
		}()
		batch, err := db.ChConn().PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", Erc20Transfer{}.TableName()))
		if err != nil {
			return errors.Wrap(err, "failed to prepare batch")
		}
		for _, e := range b.erc20Transfer {
			if err := batch.AppendStruct(e); err != nil {
				return errors.Wrap(err, "failed to append struct")
			}
		}
		if err := batch.Send(); err != nil {
			return errors.Wrap(err, "failed to send batch")
		}
	}

	if len(b.erc20Approval) > 0 {
		defer func() {
			b.erc20Approval = b.erc20Approval[:0]
		}()
		batch, err := db.ChConn().PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", Erc20Approval{}.TableName()))
		if err != nil {
			return errors.Wrap(err, "failed to prepare batch")
		}
		for _, e := range b.erc20Approval {
			if err := batch.AppendStruct(e); err != nil {
				return errors.Wrap(err, "failed to append struct")
			}
		}
		if err := batch.Send(); err != nil {
			return errors.Wrap(err, "failed to send batch")
		}
	}

	if len(b.erc20Holder) > 0 {
		defer func() {
			b.erc20Holder = b.erc20Holder[:0]
		}()
		batch, err := db.ChConn().PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", Erc20Holder{}.TableName()))
		if err != nil {
			return errors.Wrap(err, "failed to prepare batch")
		}
		for _, e := range b.erc20Holder {
			if err := batch.AppendStruct(e); err != nil {
				return errors.Wrap(err, "failed to append struct")
			}
		}
		if err := batch.Send(); err != nil {
			return errors.Wrap(err, "failed to send batch")
		}
	}

	return db.UpdateIndexHeight(b.Name(), b.tipHeight)
}

func (b tokenPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b tokenPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = tokenPlugin{}
