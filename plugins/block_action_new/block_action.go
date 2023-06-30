package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.3.0"

var (
	queue      = []*models.BlockActionNew{}
	updateTime = time.Now()
)

type blockActionNewPlugin struct {
}

func (b blockActionNewPlugin) Name() string {
	return "block_action_new"
}

func (b blockActionNewPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockActionNewPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(),
		&models.BlockActionNew{},
	); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b blockActionNewPlugin) process() error {
	if updateTime.Add(4 * time.Second).After(time.Now()) {
		return nil
	}
	var acts []models.BlockActionNew
	var blkMaxHeight uint64

	for _, act := range queue {
		acts = append(acts, *act)
		if act.BlockHeight > blkMaxHeight {
			blkMaxHeight = act.BlockHeight
		}
	}
	if len(acts) == 0 {
		return nil
	}
	if err := db.DB().CreateInBatches(&acts, 100).Error; err != nil {
		return err
	}
	if err := db.UpdateIndexHeight(b.Name(), blkMaxHeight); err != nil {
		return err
	}
	queue = []*models.BlockActionNew{}
	updateTime = time.Now()
	return nil
}

func getReceiptsFromBlock(blk *block.Block) map[hash.Hash256]*action.Receipt {
	receipts := make(map[hash.Hash256]*action.Receipt, len(blk.Receipts))
	for _, receipt := range blk.Receipts {
		receipts[receipt.ActionHash] = receipt
	}
	return receipts
}

func getPayloadAmount(act action.Action) (*big.Int, []byte) {
	amount := big.NewInt(0)

	var payload []byte
	switch a := act.(type) {
	case *action.Transfer:
		amount = a.Amount()
		payload = a.Payload()
	case *action.Execution:
		amount = a.Amount()
	case *action.DepositToRewardingFund:
		amount = a.Amount()
	case *action.ClaimFromRewardingFund:
		amount = a.Amount()
	case *action.CreateStake:
		amount = a.Amount()
		payload = a.Payload()
	case *action.DepositToStake:
		amount = a.Amount()
		payload = a.Payload()
	case *action.CandidateRegister:
		amount = a.Amount()
		payload = a.Payload()
	}
	return amount, payload
}

func processMap(totalMap map[string]int, tx *gorm.DB) error {
	for account, count := range totalMap {
		m := &models.AccountActionCount{
			Address:     account,
			ActionCount: uint64(count),
		}
		if err := m.AddCount(tx, uint64(count), models.AccountActionCountAction); err != nil {
			return err
		}
	}
	return nil
}

func getAccounts(selp action.SealedEnvelope, receipt *action.Receipt) (address.Address, string, []string, error) {
	accounts := []string{}
	sender, _ := address.FromBytes(selp.SrcPubkey().Hash())
	if sender.String() != "" {
		accounts = appendIfMissing(accounts, sender.String())
	}

	dst, _ := selp.Destination()
	if len(dst) > 0 {
		if addr, err := address.FromString(dst); err != nil {
			return sender, dst, accounts, errors.Wrapf(err, "failed to parse recipient %s", dst)
		} else {
			dst = addr.String()
		}
		accounts = append(accounts, dst)
	}

	if receipt.ContractAddress != "" {
		accounts = appendIfMissing(accounts, receipt.ContractAddress)
	}
	return sender, dst, accounts, nil
}

func (b blockActionNewPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	receipts := getReceiptsFromBlock(blk)
	for _, selp := range blk.Actions {
		actionHash, _ := selp.Hash()
		receipt, ok := receipts[actionHash]
		if !ok {
			continue
		}
		sender, _ := address.FromBytes(selp.SrcPubkey().Hash())
		dst, _ := selp.Destination()
		//sender, dst, accounts, err := getAccounts(selp, receipt)
		// if err != nil {
		// 	return errors.Wrapf(err, "failed to get accounts from action %s", actionHash)
		// }

		gasPrice := decimal.NewFromBigInt(selp.GasPrice(), 0)
		gasLimit := selp.GasLimit()
		nonce := selp.Nonce()

		act := selp.Action()
		actionType := getActionTypeString(act)
		amount, payload := getPayloadAmount(act)

		amountDec := decimal.NewFromBigInt(amount, 0)
		m := &models.BlockActionNew{
			ActionHash:         hex.EncodeToString(actionHash[:]),
			ActionType:         actionType,
			BlockHeight:        blk.Height(),
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
		}
		queue = append(queue, m)
		// if err := tx.Create(m).Error; err != nil {
		// 	return err
		// }

	}
	return b.process()
}

func (b blockActionNewPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockActionNewPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockActionNewPlugin{}
