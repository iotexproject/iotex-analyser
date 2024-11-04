package main

import (
	"context"
	"strconv"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/state"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.0.0"

var (
	ActiveBlockProducers []string
)

type blockMetaV2Plugin struct {
}

func (b blockMetaV2Plugin) Name() string {
	return "block_meta_v2"
}

func (b blockMetaV2Plugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockMetaV2Plugin) DependentPlugins() []string {
	return []string{"candidate"}
}

func (b blockMetaV2Plugin) Start(ctx context.Context) error {

	if err := db.AutoMigrate(b.Name(), &models.BlockMetaV2{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b blockMetaV2Plugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	chainClient := kernel.ChainClient()
	if blkHeight == epochHeight || ActiveBlockProducers == nil {
		if err := b.updateActiveBlockProducers(chainClient, epochNum); err != nil {
			return err
		}
	}
	grantRewardActs := make(map[hash.Hash256]bool)
	// log action index
	for _, selp := range blk.Actions {
		if _, ok := selp.Action().(*action.GrantReward); ok {
			actionHash, _ := selp.Hash()
			grantRewardActs[actionHash] = true
		}
	}
	// log receipt index
	blockReward, epochReward, foundationBonus, gasConsumed, err := getReward(blk, grantRewardActs)
	if err != nil {
		return err
	}
	producerName, producerOwnerAddress := getCandidateName(blkHeight, blk.ProducerAddress())
	var expectedProducerOwnerAddress, expectedProducerName string
	if len(ActiveBlockProducers) > 0 {
		expectedProducerAddr := ActiveBlockProducers[int(blkHeight)%len(ActiveBlockProducers)]
		if blk.ProducerAddress() == expectedProducerAddr {
			expectedProducerName = producerName
			expectedProducerOwnerAddress = producerOwnerAddress
		} else {
			expectedProducerName, expectedProducerOwnerAddress = getCandidateName(blkHeight, expectedProducerAddr)
		}
	}

	blockSize, err := getBlockSize(blk)
	if err != nil {
		return err
	}

	err = db.DB().Transaction(func(tx *gorm.DB) error {
		bm := models.BlockMetaV2{
			BlockHeight:             blkHeight,
			GasConsumed:             gasConsumed,
			ProducerName:            producerName,
			ProducerAddress:         producerOwnerAddress,
			ExpectedProducerName:    expectedProducerName,
			ExpectedProducerAddress: expectedProducerOwnerAddress,
			BlockReward:             decimal.NewFromBigInt(blockReward, 0),
			EpochReward:             decimal.NewFromBigInt(epochReward, 0),
			FoundationBonus:         decimal.NewFromBigInt(foundationBonus, 0),
			EpochNum:                epochNum,
			EpochHeight:             epochHeight,
			BlockSize:               blockSize,
		}

		if err := tx.Create(&bm).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err
}

func (b blockMetaV2Plugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockMetaV2Plugin) Version() string {
	return VERSION
}

func (b blockMetaV2Plugin) updateActiveBlockProducers(chainClient iotexapi.APIServiceClient, epochNumber uint64) error {
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
var Plugin = blockMetaV2Plugin{}
