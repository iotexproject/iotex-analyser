package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	slog "github.com/iotexproject/iotex-core/pkg/log"
)

const VERSION = "2.0.0"

var preDelete bool

type clickhouseV1Plugin struct {
	batchSize       int
	tipHeight       uint64
	blocks          []*BlockV1
	actions         []*ActionV1
	logs            []*LogV1
	transactionLogs []*TransactionLogV1
	accountIncome   []*AccountIncomeV1
}

func (b clickhouseV1Plugin) Name() string {
	return "clickhouse_v1"
}

func (b clickhouseV1Plugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b clickhouseV1Plugin) BatchSize() uint64 {
	return uint64(b.batchSize)
}

func (b *clickhouseV1Plugin) Start(ctx context.Context) error {
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
	if err = openChConn(ctx, cfg); err != nil {
		return errors.Wrapf(err, "failed to connect to clickhouse")
	}
	if height, err := AutoMigrate(b.Name(), &BlockV1{}, &ActionV1{}, &LogV1{}, &TransactionLogV1{}, &AccountIncomeV1{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	} else if height == 0 {
		initBalances := make(map[string]string)
		for addr, amount := range config.Default.Genesis.Account.InitBalanceMap {
			initBalances[addr] = amount
		}
		initBalances["io0000000000000000000000rewardingprotocol"] = config.Default.Genesis.Rewarding.InitBalanceStr
		for addr, amount := range initBalances {

			insertData := map[string]interface{}{
				"block_height": uint64(0),
				"in_flow":      amount,
				"address":      addr,
			}
			if err := chDB.Model(&AccountIncomeV1{}).Create(insertData).Error; err != nil {
				return err
			}
		}
	}
	preDelete = cfg.PreDelete
	b.batchSize = cfg.BatchSize
	return nil
}

func (b clickhouseV1Plugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[0].Height() + uint64(len(blks))
	return b.commit()
}

func (b clickhouseV1Plugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *clickhouseV1Plugin) putBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	blkHash := blk.HashBlock()
	epochNum := kernel.GetEpochNum(blkHeight)
	txtRoot := blk.Header.TxRoot()
	preHash := blk.PrevHash()
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

	b.blocks = append(b.blocks, &BlockV1{
		BlockHeight:     blk.Height(),
		BlockHash:       hex.EncodeToString(blkHash[:]),
		ProducerAddress: blk.ProducerAddress(),
		TxRoot:          hex.EncodeToString(txtRoot[:]),
		Version:         blk.Version(),
		PrevBlockHash:   hex.EncodeToString(preHash[:]),
		GasConsumed:     gasConsumed,
		BlockReward:     decimal.NewFromBigInt(blockReward, 0),
		EpochReward:     decimal.NewFromBigInt(epochReward, 0),
		FoundationBonus: decimal.NewFromBigInt(foundationBonus, 0),
		NumActions:      len(blk.Actions),
		Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
		EpochNumber:     epochNum,
	})

	receipts := getReceiptsFromBlock(blk)
	for _, selp := range blk.Actions {
		actionHash, _ := selp.Hash()
		receipt, ok := receipts[actionHash]
		if !ok {
			continue
		}
		sender, dst, err := getAccount(selp, receipt)
		if err != nil {
			return errors.Wrapf(err, "failed to get accounts from action %s", actionHash)
		}

		gasPrice := decimal.NewFromBigInt(selp.GasPrice(), 0)
		gasLimit := selp.GasLimit()
		nonce := selp.Nonce()

		act := selp.Action()
		actionType := getActionTypeString(act)
		amount, payload := getPayloadAmount(act)

		amountDec := decimal.NewFromBigInt(amount, 0)
		b.actions = append(b.actions, &ActionV1{
			BlockHeight:        blk.Height(),
			ActionHash:         hex.EncodeToString(actionHash[:]),
			ActionType:         actionType,
			Sender:             sender.String(),
			Recipient:          dst,
			GasPrice:           gasPrice,
			GasLimit:           gasLimit,
			Nonce:              nonce,
			Amount:             amountDec,
			GasConsumed:        receipt.GasConsumed,
			ChainID:            selp.ChainID(),
			Encoding:           selp.Encoding(),
			Version:            selp.Version(),
			ContractAddress:    receipt.ContractAddress,
			Status:             receipt.Status,
			Timestamp:          time.Unix(blk.Timestamp().Unix(), 0),
			ExecutionRevertMsg: strings.ReplaceAll(receipt.ExecutionRevertMsg(), string([]byte{0x00}), "0x00"),
			Payload:            payload,
		})
	}
	var incomes = make(map[string]income)
	for _, receipt := range receipts {
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		brl, err := handleLogs(receipt.Logs(), actionHash, blk.Height(), blk.Timestamp().Unix())
		if err != nil {
			return err
		}
		b.logs = append(b.logs, brl...)
		brt, err := handleTransactionLogs(receipt.TransactionLogs(), actionHash, blk.Height(), blk.Timestamp().Unix())
		if err != nil {
			return err
		}
		b.transactionLogs = append(b.transactionLogs, brt...)
		for _, transation := range receipt.TransactionLogs() {
			if transation.Sender != "" {
				inTran, ok := incomes[transation.Sender]
				if !ok {
					inTran = income{
						outFlow:       big.NewInt(0).Set(transation.Amount),
						outNumActions: 1,
						inFlow:        big.NewInt(0),
						inNumActions:  0,
					}
				} else {
					inTran.outFlow = inTran.outFlow.Add(inTran.outFlow, transation.Amount)
					inTran.outNumActions++
				}
				incomes[transation.Sender] = inTran
			}
			recipient := transation.Recipient
			if len(recipient) > 0 {
				if addr, err := address.FromString(recipient); err != nil {
					//skip invalid address
					continue
				} else {
					recipient = addr.String()
				}
			}
			if recipient != "" {
				inTran, ok := incomes[recipient]
				if !ok {
					inTran = income{
						inFlow:        big.NewInt(0).Set(transation.Amount),
						inNumActions:  1,
						outFlow:       big.NewInt(0),
						outNumActions: 0,
					}
				} else {
					inTran.inFlow = inTran.inFlow.Add(inTran.inFlow, transation.Amount)
					inTran.inNumActions++
				}
				incomes[recipient] = inTran
			}
		}
	}
	for accountAddress, accountIncome := range incomes {
		inFlow := decimal.NewFromBigInt(accountIncome.inFlow, 0)
		outFlow := decimal.NewFromBigInt(accountIncome.outFlow, 0)
		b.accountIncome = append(b.accountIncome, &AccountIncomeV1{
			BlockHeight:   blkHeight,
			Address:       accountAddress,
			InFlow:        inFlow,
			InNumActions:  accountIncome.inNumActions,
			OutFlow:       outFlow,
			OutNumActions: accountIncome.outNumActions,
			Timestamp:     time.Unix(blk.Timestamp().Unix(), 0),
		})
	}
	return nil
}

func (b *clickhouseV1Plugin) commit() error {
	fmt.Println("b.BatchSize()")
	fmt.Println(b.BatchSize())
	if len(b.blocks) > 0 {
		if err := chDB.Model(&BlockV1{}).CreateInBatches(b.blocks, b.batchSize*2).Error; err != nil {
			return err
		}
	}
	if len(b.actions) > 0 {
		if err := chDB.Model(&ActionV1{}).CreateInBatches(b.actions, b.batchSize*2).Error; err != nil {
			return err
		}
	}
	if len(b.logs) > 0 {
		if err := chDB.Model(&LogV1{}).CreateInBatches(b.logs, b.batchSize*2).Error; err != nil {
			return err
		}
	}
	if len(b.transactionLogs) > 0 {
		if err := chDB.Model(&TransactionLogV1{}).CreateInBatches(b.transactionLogs, b.batchSize*2).Error; err != nil {
			return err
		}
	}
	if len(b.actions) > 0 {
		if err := chDB.Model(&AccountIncomeV1{}).CreateInBatches(b.accountIncome, b.batchSize*2).Error; err != nil {
			return err
		}
	}
	return db.UpdateIndexHeight(b.Name(), b.tipHeight)
}

func (b clickhouseV1Plugin) Stop(ctx context.Context) error {
	return nil
}

func (b clickhouseV1Plugin) Version() string {
	return VERSION
}

func handleLogs(logs []*action.Log, actionHash string, blkHeight uint64, blkTime int64) ([]*LogV1, error) {
	var brls []*LogV1
	for _, log := range logs {
		topic0, topic1, topic2, topic3 := parseTopics(log.Topics)
		logData := log.Data
		if logData == nil {
			logData = []byte("")
		}
		brls = append(brls, &LogV1{
			BlockHeight:     blkHeight,
			ActionHash:      actionHash,
			ContractAddress: log.Address,
			Topic0:          topic0,
			Topic1:          topic1,
			Topic2:          topic2,
			Topic3:          topic3,
			Data:            logData,
			Index:           uint(log.Index),
			TxIndex:         uint(log.TxIndex),
			Timestamp:       time.Unix(blkTime, 0),
		})
	}
	return brls, nil
}

func handleTransactionLogs(transactionLogs []*action.TransactionLog, actionHash string, blkHeight uint64, blkTime int64) ([]*TransactionLogV1, error) {
	var brts []*TransactionLogV1
	for _, transation := range transactionLogs {
		transation := transation
		amountDec := decimal.NewFromBigInt(transation.Amount, 0)
		recipient := transation.Recipient
		if len(recipient) > 0 {
			if addr, err := address.FromString(recipient); err != nil {
				//skip invalid address
				continue
			} else {
				recipient = addr.String()
			}
		}
		brts = append(brts, &TransactionLogV1{
			BlockHeight: blkHeight,
			ActionHash:  actionHash,
			Type:        getActionType(transation.Type),
			Amount:      amountDec,
			Internal:    isContractAddress(transation.Sender),
			Sender:      transation.Sender,
			Recipient:   recipient,
			Timestamp:   time.Unix(blkTime, 0),
		})
	}

	return brts, nil
}

// exported
var Plugin = clickhouseV1Plugin{}
