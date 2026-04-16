package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
)

const VERSION = "2.1.2"

const (
	StakingProtocolAddress         = "io1qnpz47hx5q6r3w876axtrn6yz95d70cjl35r53"
	errBucketSumAmount             = "getBucketSumAmountFromCacheByBucketID error, bucketID: %d"
	errBucketInfoAddressByBucketID = "getBucketInfoAddressFromCacheByBucketID error"
)

type stakingActionChPlugin struct {
	batchSize         int
	tipHeight         uint64
	stakingActions    []*StakingActions
	bucketStateCache  *bucketStateCache
	pendingBucketInfo map[uint64]*pendingBucketState
}

func (b stakingActionChPlugin) Name() string {
	return "staking_actions_ch"
}

func (b stakingActionChPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b stakingActionChPlugin) BatchSize() int {
	return b.batchSize
}

func (b stakingActionChPlugin) DependentPlugins() []string {
	return []string{"candidate"}
}

func (b *stakingActionChPlugin) Start(ctx context.Context) error {
	var err error
	cfg := &Config{
		DSN:                  "tcp://127.0.0.1:8321",
		BucketStateCacheSize: defaultBucketStateCacheSize,
	}
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		if err = yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}
	if err = openChConn(ctx, cfg); err != nil {
		return errors.Wrapf(err, "failed to connect to clickhouse")
	}

	b.batchSize = cfg.BatchSize
	b.bucketStateCache = newBucketStateCache(cfg.BucketStateCacheSize)
	b.pendingBucketInfo = make(map[uint64]*pendingBucketState)
	return nil
}

func (b *stakingActionChPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	b.ensureBucketStateCache()
	if err := b.preloadBucketStates(collectBucketIDs(blks)); err != nil {
		return err
	}
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[0].Height() + uint64(len(blks)) - 1
	return b.commit()
}

func (b *stakingActionChPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	b.ensureBucketStateCache()
	if err := b.preloadBucketStates(collectBucketIDs([]*block.Block{blk})); err != nil {
		return err
	}
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *stakingActionChPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	actions := make(map[hash.Hash256]*action.SealedEnvelope, len(blk.Actions))
	bucketMap := make(map[string]uint64)
	for _, selp := range blk.Actions {
		actionHash, _ := selp.Hash()
		actions[actionHash] = selp
	}
	for index, receipt := range blk.Receipts {

		if receipt.Status != uint64(iotextypes.ReceiptStatus_Success) {
			continue
		}
		selp, ok := actions[receipt.ActionHash]
		if !ok {
			continue
		}

		sender, _ := address.FromBytes(selp.SrcPubkey().Hash())
		act := selp.Action()
		actionHash, _ := selp.Hash()
		actHash := hex.EncodeToString(actionHash[:])
		//cmpNum := big.NewInt(100000000)
		for _, log := range receipt.Logs() {
			if log.Address == StakingProtocolAddress && len(log.Topics) > 1 {
				bucketIndex := new(big.Int).SetBytes(log.Topics[1][:])

				//if bucketIndex.Cmp(cmpNum) > 0 {
				//	continue
				//}
				bucketMap[actHash] = bucketIndex.Uint64()
			}
		}
		switch a := act.(type) {
		case *action.CreateStake:
			cadidateAddr, err := getCandidateAddressByName(a.Candidate(), blk.Height())
			if err != nil {
				return err
			}

			bucketID, ok := bucketMap[actHash]
			if !ok {
				return errors.New("can not found bucketID with actHash:" + actHash)
			}

			b.appendStakingActions(&StakingActions{
				BlockHeight:  blk.Height(),
				Index:        index,
				BucketID:     bucketID,
				Sender:       sender.String(),
				OwnerAddress: sender.String(),
				ActHash:      actHash,
				Candidate:    cadidateAddr,
				Amount:       decimal.NewFromBigInt(a.Amount(), 0),
				ActType:      "StakeCreate",
				AutoStake:    a.AutoStake(),
				Duration:     a.Duration(),
				Timestamp:    time.Unix(blk.Timestamp().Unix(), 0),
			})
		case *action.TransferStake:
			bucketID := a.BucketIndex()
			decmailAmount, err := b.getBucketSumAmountFromCacheByBucketID(bucketID)
			if err != nil {
				return errors.Wrapf(err, errBucketSumAmount, bucketID)
			}
			info, err := b.getBucketInfoAddressFromCacheByBucketID(bucketID)
			if err != nil {
				return errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			b.appendStakingActions(&StakingActions{
				BlockHeight:  blk.Height(),
				Index:        index,
				BucketID:     bucketID,
				OwnerAddress: info.OwnerAddress,
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "TransferStake",
				Duration:     info.Duration,
				Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
				Timestamp:    time.Unix(blk.Timestamp().Unix(), 0),
			}, &StakingActions{
				BlockHeight:  blk.Height(),
				Index:        index,
				BucketID:     bucketID,
				OwnerAddress: a.VoterAddress().String(),
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "TransferStake",
				Duration:     info.Duration,
				Amount:       decmailAmount,
				Timestamp:    time.Unix(blk.Timestamp().Unix(), 0),
			})
		case *action.Restake:
			bucketID := a.BucketIndex()
			info, err := b.getBucketInfoAddressFromCacheByBucketID(bucketID)
			if err != nil {
				return errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			// fix greenland (height=6544441) restake
			fixAmount := decimal.NewFromInt(0)
			if blk.Height() < genesis.Default.GreenlandBlockHeight {
				fixAmount, err = b.getFixBucketSumAmountFromCacheByBucketID(bucketID)
				if err != nil {
					return errors.Wrapf(err, errBucketSumAmount, bucketID)
				}
			}
			b.appendStakingActions(&StakingActions{
				BlockHeight:  blk.Height(),
				Index:        index,
				BucketID:     bucketID,
				OwnerAddress: sender.String(),
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    a.AutoStake(),
				ActType:      "Restake",
				Duration:     a.Duration(),
				Amount:       fixAmount,
				Timestamp:    time.Unix(blk.Timestamp().Unix(), 0),
			})
		case *action.ChangeCandidate:
			bucketID := a.BucketIndex()
			decmailAmount, err := b.getBucketSumAmountFromCacheByBucketID(bucketID)
			if err != nil {
				return errors.Wrapf(err, errBucketSumAmount, bucketID)
			}
			info, err := b.getBucketInfoAddressFromCacheByBucketID(bucketID)
			if err != nil {
				return errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			b.appendStakingActions(&StakingActions{
				BlockHeight:  blk.Height(),
				Index:        index,
				BucketID:     bucketID,
				OwnerAddress: info.OwnerAddress,
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "ChangeCandidate",
				Duration:     info.Duration,
				Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
				Timestamp:    time.Unix(blk.Timestamp().Unix(), 0),
			})
			cadidateAddr, err := getCandidateAddressByName(a.Candidate(), blk.Height())
			if err != nil {
				return err
			}
			b.appendStakingActions(&StakingActions{
				BlockHeight:  blk.Height(),
				Index:        index,
				BucketID:     bucketID,
				OwnerAddress: info.OwnerAddress,
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    cadidateAddr,
				AutoStake:    info.AutoStake,
				ActType:      "ChangeCandidate",
				Duration:     info.Duration,
				Amount:       decmailAmount,
				Timestamp:    time.Unix(blk.Timestamp().Unix(), 0),
			})
			//TODO: candidate update has no bucketID
		case *action.CandidateUpdate:
			continue
		case *action.DepositToStake:
			bucketID := a.BucketIndex()
			info, err := b.getBucketInfoAddressFromCacheByBucketID(bucketID)
			if err != nil {
				return errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			b.appendStakingActions(&StakingActions{
				BlockHeight:  blk.Height(),
				Index:        index,
				BucketID:     bucketID,
				OwnerAddress: info.OwnerAddress,
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "DepositToStake",
				Duration:     info.Duration,
				Amount:       decimal.NewFromBigInt(a.Amount(), 0),
				Timestamp:    time.Unix(blk.Timestamp().Unix(), 0),
			})
		case *action.Unstake:
			bucketID := a.BucketIndex()
			decmailAmount, err := b.getBucketSumAmountFromCacheByBucketID(bucketID)
			if err != nil {
				return errors.Wrap(err, "getBucketSumAmountFromCacheByBucketID error")
			}
			info, err := b.getBucketInfoAddressFromCacheByBucketID(bucketID)
			if err != nil {
				return errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			b.appendStakingActions(&StakingActions{
				BlockHeight:  blk.Height(),
				Index:        index,
				BucketID:     bucketID,
				OwnerAddress: sender.String(),
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "Unstake",
				Duration:     info.Duration,
				Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
				Timestamp:    time.Unix(blk.Timestamp().Unix(), 0),
			})
		case *action.CandidateRegister:
			bucketID, ok := bucketMap[actHash]
			if !ok {
				return errors.New("can not found bucketID with actHash:" + actHash)
			}
			b.appendStakingActions(&StakingActions{
				BlockHeight:  blk.Height(),
				Index:        index,
				BucketID:     bucketID,
				Sender:       sender.String(),
				OwnerAddress: a.OwnerAddress().String(),
				ActHash:      actHash,
				Candidate:    a.OwnerAddress().String(),
				Amount:       decimal.NewFromBigInt(a.Amount(), 0),
				ActType:      "CandidateRegister",
				AutoStake:    a.AutoStake(),
				Duration:     a.Duration(),
				Timestamp:    time.Unix(blk.Timestamp().Unix(), 0),
			})
		}
	}

	return nil
}

func (b *stakingActionChPlugin) appendStakingActions(actions ...*StakingActions) {
	b.stakingActions = append(b.stakingActions, actions...)
	b.recordPendingStakingAction(actions...)
}

func (b *stakingActionChPlugin) commit() error {
	if len(b.stakingActions) > 0 {
		if err := chDB.Model(&StakingActions{}).CreateInBatches(b.stakingActions, len(b.stakingActions)+1).Error; err != nil {
			slog.L().Error("put stakingActions ", zap.String("plugin", b.Name()), zap.Int("stakingActions", len(b.stakingActions)))
			b.stakingActions = b.stakingActions[:0]
			b.pendingBucketInfo = make(map[uint64]*pendingBucketState)
			return err
		}
		b.applyPendingBucketStates()
		b.stakingActions = b.stakingActions[:0]
		b.pendingBucketInfo = make(map[uint64]*pendingBucketState)
	}
	return db.UpdateIndexHeight(b.Name(), b.tipHeight)
}

func (b stakingActionChPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b stakingActionChPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = &stakingActionChPlugin{}
