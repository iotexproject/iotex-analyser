package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const VERSION = "2.1.2"

const (
	// h := hash.Hash160b([]byte("staking"))
	// stakingProtocolAddr, err := address.FromBytes(h[:])
	// if err != nil {
	// 	return err
	// }
	StakingProtocolAddress         = "io1qnpz47hx5q6r3w876axtrn6yz95d70cjl35r53"
	errBucketSumAmount             = "getBucketSumAmountByBucketID error, bucketID: %d"
	errBucketInfoAddressByBucketID = "getBucketInfoAddressByBucketID error"
)

var unSelfStake *big.Int

type stakingActionPlugin struct {
	cfg Config
}

type Config struct {
	BatchSize int `yaml:"batchSize"`
}

type stash struct {
	sumAmount map[string]big.Int
	info      map[string]*BucketInfo
}

func (b stakingActionPlugin) Name() string {
	return "staking_actions"
}

func (b stakingActionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b stakingActionPlugin) BatchSize() int {
	return b.cfg.BatchSize
}

func (b stakingActionPlugin) DependentPlugins() []string {
	return []string{"candidate"}
}

func (b *stakingActionPlugin) Start(ctx context.Context) error {
	var err error
	cfg := &Config{
		BatchSize: 200,
	}
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		if err = yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}
	b.cfg = *cfg

	if err := db.ChConn().Exec(ctx, models.StakingActionsDDL); err != nil {
		return errors.Wrapf(err, "failed to create table %s", models.StakingActions{}.TableName())
	}

	var ok bool
	unSelfStake, ok = new(big.Int).SetString("000000000000000000000000000000000000000000000000ffffffffffffffff", 16)
	if !ok {
		return errors.New("can not convert string to bigint with plugin %s:" + b.Name())
	}

	return nil
}

func (b stakingActionPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	stakingActions := make([]*models.StakingActions, 0)
	stash := &stash{
		sumAmount: make(map[string]big.Int),
		info:      make(map[string]*BucketInfo),
	}
	for _, blk := range blks {
		actions, err := b.handleBlock(ctx, blk, stash)
		if err != nil {
			return errors.Wrapf(err, "failed to handle block %d", blk.Height())
		}
		stakingActions = append(stakingActions, actions...)
	}
	return b.commit(ctx, stakingActions, blks[len(blks)-1].Height())
}

func (b stakingActionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	stash := &stash{
		sumAmount: make(map[string]big.Int),
		info:      make(map[string]*BucketInfo),
	}
	stakingActions, err := b.handleBlock(ctx, blk, stash)
	if err != nil {
		return errors.Wrapf(err, "failed to handle block %d", blk.Height())
	}
	return b.commit(ctx, stakingActions, blk.Height())
}

func (b stakingActionPlugin) handleBlock(ctx context.Context, blk *block.Block, stash *stash) ([]*models.StakingActions, error) {
	var (
		stakingActions []*models.StakingActions
		stakingAction  *models.StakingActions
		logIndex       uint32
	)
	tx := db.ChConn()
	actions := make(map[hash.Hash256]*action.SealedEnvelope, len(blk.Actions))
	bucketMap := make(map[string]uint64)
	for _, selp := range blk.Actions {
		actionHash, _ := selp.Hash()
		actions[actionHash] = selp
	}
	for _, receipt := range blk.Receipts {

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
		// cmpNum := big.NewInt(100000000)
		for _, log := range receipt.Logs() {
			if log.Address == StakingProtocolAddress && len(log.Topics) > 1 {
				bucketIndex := new(big.Int).SetBytes(log.Topics[1][:])

				// if bucketIndex.Cmp(cmpNum) > 0 {
				// 	continue
				// }
				bucketMap[actHash] = bucketIndex.Uint64()
			}
		}
		switch a := act.(type) {
		case *action.CreateStake:
			cadidateAddr, err := getCandidateAddressByName(a.Candidate(), blk.Height())
			if err != nil {
				return nil, err
			}

			bucketID, ok := bucketMap[actHash]
			if !ok {
				return nil, errors.New("can not found bucketID with actHash:" + actHash)
			}
			stakingAction = &models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     strconv.FormatUint(bucketID, 10),
				Sender:       sender.String(),
				OwnerAddress: sender.String(),
				ActHash:      actHash,
				Candidate:    cadidateAddr,
				Amount:       a.Amount().String(),
				ActType:      "StakeCreate",
				AutoStake:    a.AutoStake(),
				Duration:     a.Duration(),
				LogIndex:     logIndex,
			}
			logIndex++
			stakingActions = append(stakingActions, stakingAction)
			stash.info[strconv.FormatUint(bucketID, 10)] = &BucketInfo{
				OwnerAddress: sender.String(),
				Candidate:    cadidateAddr,
				AutoStake:    a.AutoStake(),
				Duration:     a.Duration(),
			}
			sa := stash.sumAmount[strconv.FormatUint(bucketID, 10)]
			stash.sumAmount[strconv.FormatUint(bucketID, 10)] = *new(big.Int).Add(&sa, a.Amount())
		case *action.TransferStake:
			bucketID := a.BucketIndex()
			decmailAmount, err := bucketSumAmount(ctx, tx, strconv.FormatUint(bucketID, 10), stash)
			if err != nil {
				return nil, errors.Wrapf(err, errBucketSumAmount, bucketID)
			}
			info, err := bucketInfo(ctx, tx, strconv.FormatUint(bucketID, 10), stash)
			if err != nil {
				return nil, errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			stakingAction = &models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     strconv.FormatUint(bucketID, 10),
				OwnerAddress: info.OwnerAddress,
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "TransferStake",
				Duration:     info.Duration,
				Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)).String(),
				LogIndex:     logIndex,
			}
			logIndex++
			stakingActions = append(stakingActions, stakingAction)
			stakingAction = &models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     strconv.FormatUint(bucketID, 10),
				OwnerAddress: a.VoterAddress().String(),
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "TransferStake",
				Duration:     info.Duration,
				Amount:       decmailAmount.String(),
				LogIndex:     logIndex,
			}
			logIndex++
			stakingActions = append(stakingActions, stakingAction)
			stash.info[strconv.FormatUint(bucketID, 10)] = &BucketInfo{
				OwnerAddress: a.VoterAddress().String(),
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				Duration:     info.Duration,
			}
		case *action.Restake:
			bucketID := a.BucketIndex()
			info, err := bucketInfo(ctx, tx, strconv.FormatUint(bucketID, 10), stash)
			if err != nil {
				return nil, errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			// fix greenland (height=6544441) restake
			fixAmount := decimal.NewFromInt(0)
			if blk.Height() < genesis.Default.GreenlandBlockHeight {
				fixAmount, err = getFixBucketSumAmountByBucketID(ctx, tx, strconv.FormatUint(bucketID, 10))
				if err != nil {
					return nil, errors.Wrapf(err, errBucketSumAmount, bucketID)
				}
			}
			stakingAction = &models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     strconv.FormatUint(bucketID, 10),
				OwnerAddress: sender.String(),
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    a.AutoStake(),
				ActType:      "Restake",
				Duration:     a.Duration(),
				Amount:       fixAmount.String(),
				LogIndex:     logIndex,
			}
			logIndex++
			stakingActions = append(stakingActions, stakingAction)
			stash.info[strconv.FormatUint(bucketID, 10)] = &BucketInfo{
				OwnerAddress: sender.String(),
				Candidate:    info.Candidate,
				AutoStake:    a.AutoStake(),
				Duration:     a.Duration(),
			}
			sa := stash.sumAmount[strconv.FormatUint(bucketID, 10)]
			stash.sumAmount[strconv.FormatUint(bucketID, 10)] = *new(big.Int).Add(&sa, fixAmount.BigInt())
		case *action.ChangeCandidate:
			bucketID := a.BucketIndex()
			decmailAmount, err := bucketSumAmount(ctx, tx, strconv.FormatUint(bucketID, 10), stash)
			if err != nil {
				return nil, errors.Wrapf(err, errBucketSumAmount, bucketID)
			}
			info, err := bucketInfo(ctx, tx, strconv.FormatUint(bucketID, 10), stash)
			if err != nil {
				return nil, errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			stakingAction = &models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     strconv.FormatUint(bucketID, 10),
				OwnerAddress: info.OwnerAddress,
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "ChangeCandidate",
				Duration:     info.Duration,
				Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)).String(),
				LogIndex:     logIndex,
			}
			logIndex++
			stakingActions = append(stakingActions, stakingAction)
			cadidateAddr, err := getCandidateAddressByName(a.Candidate(), blk.Height())
			if err != nil {
				return nil, err
			}
			stakingAction = &models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     strconv.FormatUint(bucketID, 10),
				OwnerAddress: info.OwnerAddress,
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    cadidateAddr,
				AutoStake:    info.AutoStake,
				ActType:      "ChangeCandidate",
				Duration:     info.Duration,
				Amount:       decmailAmount.String(),
				LogIndex:     logIndex,
			}
			logIndex++
			stakingActions = append(stakingActions, stakingAction)
			stash.info[strconv.FormatUint(bucketID, 10)] = &BucketInfo{
				OwnerAddress: info.OwnerAddress,
				Candidate:    cadidateAddr,
				AutoStake:    info.AutoStake,
				Duration:     info.Duration,
			}
			//TODO: candidate update has no bucketID
		case *action.CandidateUpdate:
			logIndex++
			continue
		case *action.DepositToStake:
			bucketID := a.BucketIndex()
			info, err := bucketInfo(ctx, tx, strconv.FormatUint(bucketID, 10), stash)
			if err != nil {
				return nil, errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			stakingAction = &models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     strconv.FormatUint(bucketID, 10),
				OwnerAddress: info.OwnerAddress,
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "DepositToStake",
				Duration:     info.Duration,
				Amount:       decimal.NewFromBigInt(a.Amount(), 0).String(),
				LogIndex:     logIndex,
			}
			logIndex++
			stakingActions = append(stakingActions, stakingAction)
			sa := stash.sumAmount[strconv.FormatUint(bucketID, 10)]
			stash.sumAmount[strconv.FormatUint(bucketID, 10)] = *new(big.Int).Add(&sa, a.Amount())
		case *action.Unstake:
			bucketID := a.BucketIndex()
			decmailAmount, err := bucketSumAmount(ctx, tx, strconv.FormatUint(bucketID, 10), stash)
			if err != nil {
				return nil, errors.Wrap(err, "getBucketSumAmountByBucketID error")
			}
			info, err := bucketInfo(ctx, tx, strconv.FormatUint(bucketID, 10), stash)
			if err != nil {
				return nil, errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			stakingAction = &models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     strconv.FormatUint(bucketID, 10),
				OwnerAddress: sender.String(),
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "Unstake",
				Duration:     info.Duration,
				Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)).String(),
				LogIndex:     logIndex,
			}
			logIndex++
			stakingActions = append(stakingActions, stakingAction)
			sa := stash.sumAmount[strconv.FormatUint(bucketID, 10)]
			stash.sumAmount[strconv.FormatUint(bucketID, 10)] = *new(big.Int).Add(&sa, decmailAmount.Mul(decimal.NewFromInt(-1)).BigInt())
		case *action.CandidateRegister:
			bucketID, ok := bucketMap[actHash]
			if !ok {
				return nil, errors.New("can not found bucketID with actHash:" + actHash)
			}

			if bucketID != unSelfStake.Uint64() {
				stakingAction = &models.StakingActions{
					BlockHeight:  blk.Height(),
					BucketID:     strconv.FormatUint(bucketID, 10),
					Sender:       sender.String(),
					OwnerAddress: a.OwnerAddress().String(),
					ActHash:      actHash,
					Candidate:    a.OwnerAddress().String(),
					Amount:       decimal.NewFromBigInt(a.Amount(), 0).String(),
					ActType:      "CandidateRegister",
					AutoStake:    a.AutoStake(),
					Duration:     a.Duration(),
					LogIndex:     logIndex,
				}
				logIndex++
				stakingActions = append(stakingActions, stakingAction)
				stash.info[strconv.FormatUint(bucketID, 10)] = &BucketInfo{
					OwnerAddress: a.OwnerAddress().String(),
					Candidate:    a.OwnerAddress().String(),
					AutoStake:    a.AutoStake(),
					Duration:     a.Duration(),
				}
				sa := stash.sumAmount[strconv.FormatUint(bucketID, 10)]
				stash.sumAmount[strconv.FormatUint(bucketID, 10)] = *new(big.Int).Add(&sa, a.Amount())
			}
		}
	}
	return stakingActions, nil
}

func (b stakingActionPlugin) commit(ctx context.Context, stakingActions []*models.StakingActions, height uint64) error {
	if len(stakingActions) > 0 {
		// batch insert to clickhouse
		batch, err := db.ChConn().PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", models.StakingActions{}.TableName()))
		if err != nil {
			return errors.Wrap(err, "failed to prepare batch")
		}
		for _, e := range stakingActions {
			if err := batch.AppendStruct(e); err != nil {
				return errors.Wrap(err, "failed to append struct")
			}
		}
		if err := batch.Send(); err != nil {
			return errors.Wrap(err, "failed to send batch")
		}
	}
	return db.UpdateIndexHeightByTx(db.DB(), b.Name(), height)
}

func (b stakingActionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b stakingActionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = stakingActionPlugin{}
