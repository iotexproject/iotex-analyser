package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	slog "github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const VERSION = "2.0.0"

var preDelete bool

type clickhousePlugin struct {
}

func (b clickhousePlugin) Name() string {
	return "clickhouse"
}

func (b clickhousePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b clickhousePlugin) Start(ctx context.Context) error {
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
	if height, err := AutoMigrate(b.Name(), &Block{}, &Action{}, &Log{}, &TransactionLog{}, &AccountIncome{}); err != nil {
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
			if err := chDB.Model(&AccountIncome{}).Create(insertData).Error; err != nil {
				return err
			}
		}
	}
	preDelete = cfg.PreDelete
	return nil
}

func (b clickhousePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
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

	block := &Block{
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
	}
	if preDelete {
		if err := chDB.Where("block_height = ?", blk.Height()).Delete(&Block{}).Error; err != nil {
			return err
		}
	}
	if err := chDB.Create(block).Error; err != nil {
		return err
	}

	receipts := getReceiptsFromBlock(blk)
	var acts []*Action
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
		acts = append(acts, &Action{
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
	if preDelete {
		if err := chDB.Where("block_height = ?", blk.Height()).Delete(&Action{}).Error; err != nil {
			return err
		}
	}
	slog.L().Info("put clickhouse actions ", zap.String("plugin", b.Name()), zap.Int("actions", len(acts)))
	if err := chDB.Model(&Action{}).CreateInBatches(acts, 200).Error; err != nil {
		return err
	}

	//logs
	var logs []*Log
	var transactionLogs []*TransactionLog
	var incomes = make(map[string]income)
	for _, receipt := range receipts {
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		brl, err := handleLogs(receipt.Logs(), actionHash, blk.Height(), blk.Timestamp().Unix())
		if err != nil {
			return err
		}
		logs = append(logs, brl...)
		brt, err := handleTransactionLogs(receipt.TransactionLogs(), actionHash, blk.Height(), blk.Timestamp().Unix())
		if err != nil {
			return err
		}
		transactionLogs = append(transactionLogs, brt...)
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
	if preDelete {
		if err := chDB.Where("block_height = ?", blk.Height()).Delete(&Log{}).Error; err != nil {
			return err
		}
	}
	slog.L().Info("put clickhouse logs ", zap.String("plugin", b.Name()), zap.Int("logs", len(logs)))
	if len(logs) > 0 {
		if err := chDB.Model(&Log{}).CreateInBatches(logs, 200).Error; err != nil {
			return err
		}
	}
	if preDelete {
		if err := chDB.Where("block_height = ?", blk.Height()).Delete(&TransactionLog{}).Error; err != nil {
			return err
		}
	}
	slog.L().Info("put clickhouse transactionLogs ", zap.String("plugin", b.Name()), zap.Int("transactionLogs", len(transactionLogs)))
	if len(transactionLogs) > 0 {
		if err := chDB.Model(&TransactionLog{}).CreateInBatches(transactionLogs, 200).Error; err != nil {
			return err
		}
	}
	if preDelete {
		if err := chDB.Where("block_height = ?", blk.Height()).Delete(&AccountIncome{}).Error; err != nil {
			return err
		}
	}
	for accountAddress, accountIncome := range incomes {
		inFlow := decimal.NewFromBigInt(accountIncome.inFlow, 0)
		outFlow := decimal.NewFromBigInt(accountIncome.outFlow, 0)
		aim := &AccountIncome{
			BlockHeight:   blkHeight,
			Address:       accountAddress,
			InFlow:        inFlow,
			InNumActions:  accountIncome.inNumActions,
			OutFlow:       outFlow,
			OutNumActions: accountIncome.outNumActions,
			Timestamp:     time.Unix(blk.Timestamp().Unix(), 0),
		}
		if err := chDB.Model(&AccountIncome{}).Create(aim).Error; err != nil {
			return err
		}
	}
	return db.UpdateIndexHeight(b.Name(), blk.Height())
}

func (b clickhousePlugin) Stop(ctx context.Context) error {
	return nil
}

func (b clickhousePlugin) Version() string {
	return VERSION
}

func handleLogs(logs []*action.Log, actionHash string, blkHeight uint64, blkTime int64) ([]*Log, error) {
	var brls []*Log
	for _, log := range logs {
		topic0, topic1, topic2, topic3 := parseTopics(log.Topics)
		logData := log.Data
		if logData == nil {
			logData = []byte("")
		}
		brls = append(brls, &Log{
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

func handleTransactionLogs(transactionLogs []*action.TransactionLog, actionHash string, blkHeight uint64, blkTime int64) ([]*TransactionLog, error) {
	var brts []*TransactionLog
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
		brts = append(brts, &TransactionLog{
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
var Plugin = clickhousePlugin{}
