package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"gorm.io/gorm"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const (
	Version = "0.0.1"
	// https://github.com/eth-infinitism/account-abstraction/blob/develop/contracts/interfaces/IEntryPoint.sol#L46-L51 event signature
	deployedHashString = "d51a9c61267aa6196961883ecf5ff2da6619c37dac0fa92122513fb32c032d2d"
	EventABI           = `[
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": true,
					"internalType": "bytes32",
					"name": "userOpHash",
					"type": "bytes32"
				},
				{
					"indexed": true,
					"internalType": "address",
					"name": "sender",
					"type": "address"
				},
				{
					"indexed": false,
					"internalType": "address",
					"name": "factory",
					"type": "address"
				},
				{
					"indexed": false,
					"internalType": "address",
					"name": "paymaster",
					"type": "address"
				}
			],
			"name": "AccountDeployed",
			"type": "event"
		}
	]`
)

var (
	deployedHash hash.Hash256
	eventABI     abi.ABI
)

type config struct {
	StartHeight uint64 `yaml:"startHeight"`
	EntryPoint  string `yaml:"entryPoint"`
}

type accountAbstractionPlugin struct {
	cfg       *config
	tipHeight uint64
	records   []*AccountAbstractionAccountDeployed
}

func (b *accountAbstractionPlugin) Name() string {
	return "account_abstraction"
}

func (b *accountAbstractionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *accountAbstractionPlugin) Start(ctx context.Context) error {
	height, err := db.GetIndexHeight(b.Name())
	if err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	deployedHash, err = hash.HexStringToHash256(deployedHashString)
	if err != nil {
		return errors.Wrap(err, "failed to decode deployed hash")
	}
	eventABI, err = abi.JSON(strings.NewReader(EventABI))
	if err != nil {
		return err
	}
	if err := db.AutoMigrate(b.Name(),
		&AccountAbstractionAccountDeployed{},
	); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	startHeight := uint64(0)
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		if bytes.Equal(cfgData, []byte{}) {
			return errors.New("plugin config is empty, please setting plugin config	`server/pluginConfigs/account_abstraction/{startHeight,entryPoint}`")
		}
		if err := yaml.Unmarshal(cfgData, b.cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			startHeight = b.cfg.StartHeight
			log.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", b.cfg))
		}
	}
	if height < startHeight && startHeight > 0 {
		return db.UpdateIndexHeight(b.Name(), startHeight-1)
	}
	return nil
}

func (b *accountAbstractionPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	for _, receipt := range blk.Receipts {
		if receipt.Status != uint64(iotextypes.ReceiptStatus_Success) {
			continue
		}
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		for _, l := range receipt.Logs() {
			if l.Address != b.cfg.EntryPoint || l.Topics[0] != deployedHash {
				continue
			}
			event := struct {
				UserOpHash common.Hash
				Sender     common.Address
				Factory    common.Address
				Paymaster  common.Address
			}{}
			err := kernel.UnpackLog(eventABI, &event, "AccountDeployed", l)
			if err != nil {
				return errors.Wrap(err, "failed to unpack AccountDeployed")
			}
			sender, _ := address.FromBytes(event.Sender.Bytes())
			factory, _ := address.FromBytes(event.Factory.Bytes())
			paymaster, _ := address.FromBytes(event.Paymaster.Bytes())
			b.records = append(b.records, &AccountAbstractionAccountDeployed{
				BlockHeight: blk.Height(),
				ActionHash:  actionHash,
				UserOpHash:  hex.EncodeToString(event.UserOpHash[:]),
				Sender:      sender.String(),
				Factory:     factory.String(),
				Paymaster:   paymaster.String(),
			})
		}
	}
	return nil
}

func (b *accountAbstractionPlugin) commit() error {
	records := b.records
	b.records = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(records) > 0 {
			if err := tx.CreateInBatches(records, 200).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *accountAbstractionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *accountAbstractionPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *accountAbstractionPlugin) BatchSize() int {
	return 1000
}

func (b *accountAbstractionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *accountAbstractionPlugin) CatchUpSafe() bool { return true }

func (b *accountAbstractionPlugin) Version() string {
	return Version
}

// exported
var Plugin = accountAbstractionPlugin{
	cfg: &config{},
}
