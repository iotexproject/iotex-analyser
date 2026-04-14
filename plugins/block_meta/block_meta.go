package main

import (
	"context"
	"strconv"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/iotexproject/iotex-core/v2/state"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

const VERSION = "2.0.7"

var (
	ActiveBlockProducers []string
)

type Config struct {
	ReviseHeight uint64 `yaml:"reviseHeight"`
}

type blockMetaPlugin struct {
	cfg       Config
	tipHeight uint64
	metas     []models.BlockMeta
}

func (b *blockMetaPlugin) Name() string {
	return "block_meta"
}

func (b *blockMetaPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *blockMetaPlugin) Start(ctx context.Context) error {

	if err := db.AutoMigrate(b.Name(), &models.BlockMeta{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	cfg := &Config{}
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		if err := yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		}
		log.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		b.cfg = *cfg
	}
	return nil
}

func (b *blockMetaPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	chainClient := kernel.ChainClient()
	if blkHeight == epochHeight || ActiveBlockProducers == nil {
		if err := b.updateActiveBlockProducers(chainClient, epochNum); err != nil {
			return err
		}
	}
	if b.cfg.ReviseHeight > 0 && blkHeight == b.cfg.ReviseHeight {
		b.reviseEpochNumber()
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
	b.metas = append(b.metas, bm)
	return nil
}

func (b *blockMetaPlugin) commit() error {
	metas := b.metas
	b.metas = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		for i := range metas {
			if err := tx.Save(&metas[i]).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *blockMetaPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *blockMetaPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *blockMetaPlugin) BatchSize() int {
	return 1000
}

func (b *blockMetaPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *blockMetaPlugin) Version() string {
	return VERSION
}

func (b *blockMetaPlugin) updateActiveBlockProducers(chainClient iotexapi.APIServiceClient, epochNumber uint64) error {
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

func (b *blockMetaPlugin) reviseEpochNumber() {
	// revise epoch number for block meta from wake block height to latest
	var (
		from = config.Default.Genesis.Blockchain.WakeBlockHeight
	)
	db.DB().Transaction(func(tx *gorm.DB) error {
		var blockMetas []models.BlockMeta
		if err := tx.Where("block_height >= ?", from).Find(&blockMetas).Error; err != nil {
			return errors.Wrap(err, "failed to get block metas with epoch_num = 0")
		}
		revisedCnt := 0
		maxHeight := from
		for _, bm := range blockMetas {
			epochNum := kernel.GetEpochNum(bm.BlockHeight)
			if bm.EpochNum == epochNum {
				continue
			}
			bm.EpochNum = epochNum
			if err := tx.Save(&bm).Error; err != nil {
				return errors.Wrapf(err, "failed to update block meta %d", bm.BlockHeight)
			}
			revisedCnt++
			if bm.BlockHeight > maxHeight {
				maxHeight = bm.BlockHeight
			}
		}
		log.L().Info("revised epoch number for block metas", zap.Uint64("from", from), zap.Int("revisedCnt", revisedCnt), zap.Uint64("max", maxHeight))
		return nil
	})
}

// exported
var Plugin = blockMetaPlugin{}
