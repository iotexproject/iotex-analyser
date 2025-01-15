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

const VERSION = "2.0.7"

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

func (b blockMetaPlugin) BatchSize() int {
	return 1000
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
	blockReward, epochReward, foundationBonus, priorityBonus, gasConsumed, err := getReward(blk, grantRewardActs)
	if err != nil {
		return err
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

	blockSize, err := getBlockSize(blk)
	if err != nil {
		return err
	}

	err = db.DB().Transaction(func(tx *gorm.DB) error {
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
			PriorityBonus:           decimal.NewFromBigInt(priorityBonus, 0),
			EpochNum:                epochNum,
			EpochHeight:             epochHeight,
			BlockSize:               blockSize,
			BlobGasUsed:             blk.BlobGasUsed(),
			ExcessBlobGas:           blk.ExcessBlobGas(),
		}
		if blk.BaseFee() != nil {
			bm.BaseFee = decimal.NewFromBigInt(blk.BaseFee(), 0)
		}

		if err := tx.Save(&bm).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err
}

func (b blockMetaPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	ms := []models.BlockMeta{}
	blkHeight := uint64(0)
	for _, blk := range blks {
		blkHeight = blk.Height()
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
		blockReward, epochReward, foundationBonus, priorityBonus, gasConsumed, err := getReward(blk, grantRewardActs)
		if err != nil {
			return err
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

		blockSize, err := getBlockSize(blk)
		if err != nil {
			return err
		}

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
			PriorityBonus:           decimal.NewFromBigInt(priorityBonus, 0),
			EpochNum:                epochNum,
			EpochHeight:             epochHeight,
			BlockSize:               blockSize,
			BlobGasUsed:             blk.BlobGasUsed(),
			ExcessBlobGas:           blk.ExcessBlobGas(),
		}
		if blk.BaseFee() != nil {
			bm.BaseFee = decimal.NewFromBigInt(blk.BaseFee(), 0)
		}
		ms = append(ms, bm)
	}
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.BlockMeta{}).CreateInBatches(ms, 200).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blkHeight)
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
