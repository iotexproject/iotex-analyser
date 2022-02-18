package main

import (
	"context"
	"math/big"
	"strconv"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/state"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.0.6"

var (
	ActiveBlockProducers []string
)

type blockMetaPlugin struct {
}

func (b blockMetaPlugin) Name() string {
	return "block_meta"
}

func (b blockMetaPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockMetaPlugin) Start(ctx context.Context) error {

	if err := db.AutoMigrate(b.Name(), &models.BlockMeta{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b blockMetaPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	chainClient := kernel.ChainClient()
	if blkHeight == epochHeight {
		if err := b.updateActiveBlockProducers(chainClient, epochNum); err != nil {
			return err
		}
	}
	var gasConsumed uint64
	grantRewardActs := make(map[hash.Hash256]bool)
	// log action index
	for _, selp := range blk.Actions {
		if _, ok := selp.Action().(*action.GrantReward); ok {
			actionHash, _ := selp.Hash()
			grantRewardActs[actionHash] = true
		}
	}
	blockReward := big.NewInt(0)
	epochReward := big.NewInt(0)
	foundationBonus := big.NewInt(0)
	// log receipt index
	for _, receipt := range blk.Receipts {
		gasConsumed += receipt.GasConsumed
		if _, ok := grantRewardActs[receipt.ActionHash]; ok {
			// Parse receipt of grant reward
			rewardInfoMap, err := getRewardInfoFromReceipt(receipt)
			if err != nil {
				return errors.Wrap(err, "failed to get reward info from receipt")
			}
			if len(rewardInfoMap) == 0 {
				continue
			}
			for _, rewards := range rewardInfoMap {
				blockReward.Add(blockReward, rewards.BlockReward)
				epochReward.Add(epochReward, rewards.EpochReward)
				foundationBonus.Add(foundationBonus, rewards.FoundationBonus)
			}
		}
	}
	producerName := getCandidateName(blkHeight, blk.ProducerAddress())
	expectedProducerAddr := ""
	expectedProducerName := ""
	if len(ActiveBlockProducers) > 0 {
		expectedProducerAddr = ActiveBlockProducers[int(blkHeight)%len(ActiveBlockProducers)]
		if blk.ProducerAddress() == expectedProducerAddr {
			expectedProducerName = producerName
		} else {
			expectedProducerName = getCandidateName(blkHeight, expectedProducerAddr)
		}
	}

	err := db.DB().Transaction(func(tx *gorm.DB) error {
		bm := models.BlockMeta{
			BlockHeight:             blkHeight,
			GasConsumed:             gasConsumed,
			ProducerName:            producerName,
			ProducerAddress:         blk.ProducerAddress(),
			ExpectedProducerName:    expectedProducerName,
			ExpectedProducerAddress: expectedProducerAddr,
			BlockReward:             decimal.NewFromBigInt(blockReward, 0),
			EpochReward:             decimal.NewFromBigInt(epochReward, 0),
			FoundationBonus:         decimal.NewFromBigInt(foundationBonus, 0),
			EpochNum:                epochNum,
			EpochHeight:             epochHeight,
		}

		if err := tx.Create(&bm).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err
}

func (b blockMetaPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockMetaPlugin) Version() string {
	return VERSION
}

func (b blockMetaPlugin) updateActiveBlockProducers(chainClient iotexapi.APIServiceClient, epochNumber uint64) error {
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte("poll"),
		MethodName: []byte("ActiveBlockProducersByEpoch"),
		Arguments:  [][]byte{[]byte(strconv.FormatUint(epochNumber, 10))},
	}
	readStateRes, err := chainClient.ReadState(context.Background(), readStateRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get active block producers")
	}

	var activeBlockProducers state.CandidateList
	if err := activeBlockProducers.Deserialize(readStateRes.GetData()); err != nil {
		return errors.Wrap(err, "failed to deserialize active block producers")
	}
	ActiveBlockProducers = []string{}
	for _, activeBlockProducer := range activeBlockProducers {
		ActiveBlockProducers = append(ActiveBlockProducers, activeBlockProducer.Address)
	}

	return nil
}

// exported
var Plugin = blockMetaPlugin{}
