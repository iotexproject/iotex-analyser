package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
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
	"gopkg.in/yaml.v2"
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
	conn      driver.Conn
	batchSize int
}

type result struct {
	erc721Transfer       []*Erc721Transfer
	erc721Approval       []*Erc721Approval
	erc721Holder         []*Erc721Holder
	erc721ApprovalForAll []*Erc721ApprovalForAll
}

func (b tokenPlugin) Name() string {
	return "erc721_" + VERSION
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) BatchSize() int {
	return b.batchSize
}

func openChConn(cfg *Config) (driver.Conn, error) {
	op, err := clickhouse.ParseDSN(cfg.DSN)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse clickhouse dsn")
	}
	chConn, err := clickhouse.Open(op)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect clickhouse")
	}
	return chConn, nil
}

func (b tokenPlugin) Start(ctx context.Context) error {
	var err error
	cfg := &Config{
		DSN: "tcp://127.0.0.1:8321",
	}
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		if err = yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}
	conn, err := openChConn(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to connect to clickhouse")
	}
	b.conn = conn
	if err := b.migrateTable(ctx); err != nil {
		return errors.Wrap(err, "failed to migrate clickhouse table")
	}

	b.batchSize = cfg.BatchSize

	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	return nil
}

func (b tokenPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	total := &result{}
	for _, blk := range blks {
		res, err := b.putBlock(ctx, blk)
		if err != nil {
			return err
		}
		total.erc721Approval = append(total.erc721Approval, res.erc721Approval...)
		total.erc721ApprovalForAll = append(total.erc721ApprovalForAll, res.erc721ApprovalForAll...)
		total.erc721Holder = append(total.erc721Holder, res.erc721Holder...)
		total.erc721Transfer = append(total.erc721Transfer, res.erc721Transfer...)
	}
	return b.commit(blks[0].Height()+uint64(len(blks))-1, total)
}

func (b tokenPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	res, err := b.putBlock(ctx, blk)
	if err != nil {
		return err
	}
	return b.commit(blk.Height(), res)
}

func (b tokenPlugin) putBlock(ctx context.Context, blk *block.Block) (*result, error) {
	res := &result{}
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
					return nil, errors.Wrap(err, "failed to check erc721")
				}
				if !ok {
					nonErc721Contract[log.Address] = struct{}{}
					continue
				}
				erc721Contract[log.Address] = struct{}{}
			}
			var holders []string
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
					return nil, errors.WithMessagef(err, "failed to unpack Transfer on action: %s", actionHash)
				}
				tokenID := decimal.NewFromBigInt(event.TokenId, 0)
				fromAddr, _ := address.FromBytes(event.From.Bytes())
				toAddr, _ := address.FromBytes(event.To.Bytes())
				model := Erc721Transfer{
					BlockHeight:     blk.Height(),
					LogIndex:        log.Index,
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					TokenId:         tokenID.String(),
					Sender:          fromAddr.String(),
					Recipient:       toAddr.String(),
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				}
				res.erc721Transfer = append(res.erc721Transfer, &model)
				holders = []string{fromAddr.String(), toAddr.String()}

			//Approval(address indexed owner, address indexed approved, uint256 indexed tokenId);
			case Approval:
				event := struct {
					Owner    common.Address
					Approved common.Address
					TokenId  *big.Int
				}{}
				err := kernel.UnpackLog(erc721ABI, &event, "Approval", log)
				if err != nil {
					return nil, errors.WithMessagef(err, "failed to unpack Approval on action: %s", actionHash)
				}
				tokenID := decimal.NewFromBigInt(event.TokenId, 0)
				owner, _ := address.FromBytes(event.Owner.Bytes())
				approved, _ := address.FromBytes(event.Approved.Bytes())
				model := Erc721Approval{
					BlockHeight:     blk.Height(),
					LogIndex:        log.Index,
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					TokenId:         tokenID.String(),
					Owner:           owner.String(),
					Approved:        approved.String(),
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				}
				res.erc721Approval = append(res.erc721Approval, &model)
			//ApprovalForAll(address indexed owner, address indexed operator, bool approved);
			case ApprovalForAll:
				event := struct {
					Owner    common.Address
					Operator common.Address
					Approved bool
				}{}
				err := kernel.UnpackLog(erc721ABI, &event, "ApprovalForAll", log)
				if err != nil {
					return nil, errors.WithMessagef(err, "ApprovalForAll event: %v", &event)
				}
				owner, _ := address.FromBytes(event.Owner.Bytes())
				operator, _ := address.FromBytes(event.Operator.Bytes())
				model := Erc721ApprovalForAll{
					BlockHeight:     blk.Height(),
					LogIndex:        log.Index,
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					Owner:           owner.String(),
					Operator:        operator.String(),
					Approved:        event.Approved,
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				}
				res.erc721ApprovalForAll = append(res.erc721ApprovalForAll, &model)
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
				model := Erc721Holder{
					ContractAddress: log.Address,
					Holder:          addr,
				}
				res.erc721Holder = append(res.erc721Holder, &model)
			}
		}
	}
	return res, nil
}

func (b tokenPlugin) commit(height uint64, res *result) error {
	if len(res.erc721Transfer) > 0 {
		batch, err := b.conn.PrepareBatch(context.Background(), "INSERT INTO erc721_transfers_v2_2_3")
		if err != nil {
			return errors.Wrap(err, "failed to prepare batch")
		}
		for _, e := range res.erc721Transfer {
			if err := batch.AppendStruct(e); err != nil {
				return errors.Wrap(err, "failed to append struct")
			}
		}
		if err := batch.Send(); err != nil {
			return errors.Wrap(err, errFailedInsertTable)
		}
	}
	if len(res.erc721Approval) > 0 {
		batch, err := b.conn.PrepareBatch(context.Background(), "INSERT INTO erc721_approvals_v2_2_3")
		if err != nil {
			return errors.Wrap(err, "failed to prepare batch")
		}
		for _, e := range res.erc721Approval {
			if err := batch.AppendStruct(e); err != nil {
				return errors.Wrap(err, "failed to append struct")
			}
		}
		if err := batch.Send(); err != nil {
			return errors.Wrap(err, errFailedInsertTable)
		}
	}
	if len(res.erc721Holder) > 0 {
		batch, err := b.conn.PrepareBatch(context.Background(), "INSERT INTO erc721_holders_v2_2_3")
		if err != nil {
			return errors.Wrap(err, "failed to prepare batch")
		}
		for _, e := range res.erc721Holder {
			if err := batch.AppendStruct(e); err != nil {
				return errors.Wrap(err, "failed to append struct")
			}
		}
		if err := batch.Send(); err != nil {
			return errors.Wrap(err, errFailedInsertTable)
		}
	}
	if len(res.erc721ApprovalForAll) > 0 {
		batch, err := b.conn.PrepareBatch(context.Background(), "INSERT INTO erc721_approval_for_alls_v2_2_3")
		if err != nil {
			return errors.Wrap(err, "failed to prepare batch")
		}
		for _, e := range res.erc721ApprovalForAll {
			if err := batch.AppendStruct(e); err != nil {
				return errors.Wrap(err, "failed to append struct")
			}
		}
		if err := batch.Send(); err != nil {
			return errors.Wrap(err, errFailedInsertTable)
		}
	}

	err := db.UpdateIndexHeight(b.Name(), height)
	return errors.Wrap(err, "failed to update index height")
}

func (b tokenPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b tokenPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = tokenPlugin{}
