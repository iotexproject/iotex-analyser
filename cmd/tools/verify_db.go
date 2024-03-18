package tools

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"strings"
	"unicode"

	"github.com/gammazero/workerpool"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/action/protocol"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/blockchain/filedao"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/schollz/progressbar/v3"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

const (
	transfer                   = "transfer"
	execution                  = "execution"
	depositToRewardingFund     = "depositToRewardingFund"
	claimFromRewardingFund     = "claimFromRewardingFund"
	stakeCreate                = "stakeCreate"
	stakeWithdraw              = "stakeWithdraw"
	stakeAddDeposit            = "stakeAddDeposit"
	candidateRegisterFee       = "candidateRegisterFee"
	candidateRegisterSelfStake = "candidateRegisterSelfStake"
	gasFee                     = "gasFee"
)

var (
	ErrActionNoMatch      = errors.New("action no match")
	ErrReceiptNoMatch     = errors.New("receipt no match")
	ErrTransactionNoMatch = errors.New("transaction no match")
)
var VerifyDB = &cli.Command{
	Name:        "verify_db",
	Usage:       "verify_db --start=<startBlkNum> --end=<endBlkNum> --worker <workerSize>",
	Description: "verify_db will verify the db by blkNum",
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "start",
			Usage: "start block number",
			Value: 1,
		},
		&cli.Uint64Flag{
			Name:  "end",
			Usage: "end block number",
			Value: 1,
		},
		&cli.IntFlag{
			Name:  "worker",
			Usage: "worker thread num",
			Value: runtime.NumCPU(),
		},
	},
	Action: verifyDB,
}

func openDAO(c *cli.Context) (blockdao.BlockDAO, error) {
	var tip protocol.TipInfo
	ctxDao := protocol.WithBlockchainCtx(
		genesis.WithGenesisContext(context.Background(), genesis.Default),
		protocol.BlockchainCtx{
			Tip: tip,
		},
	)
	var err error
	var dao blockdao.BlockDAO
	deser := block.NewDeserializer(config.EVMNetworkID())
	blockDB := config.Default.BlockDB
	blockDB.ReadOnly = true
	dao, err = filedao.NewFileDAO(blockDB, deser)
	if err != nil {
		return nil, err
	}
	if err = dao.Start(ctxDao); err != nil {
		return nil, err
	}

	return dao, nil
}

func getHeightByDB(db *gorm.DB) uint64 {
	var height uint64
	query := "SELECT min(height) FROM index_heights WHERE name in ('block_receipts','block_action','block')"
	db.Raw(query).Scan(&height)
	return height
}

func verifyDB(c *cli.Context) error {
	startBlkNum := c.Uint64("start")
	endBlkNum := c.Uint64("end")
	db, err := db.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect db")
	}
	dbHeight := getHeightByDB(db)
	if endBlkNum > dbHeight {
		endBlkNum = dbHeight
	}

	if startBlkNum > endBlkNum {
		return errors.New("start blkNum is bigger than end blkNum")
	}
	//fmt.Printf("startBlkNum=%d endBlkNum=%d dbHeight=%d \n", startBlkNum, endBlkNum, dbHeight)
	dao, err := openDAO(c)
	if err != nil {
		return errors.Wrap(err, "failed to open DAO")
	}
	defer dao.Stop(c.Context)
	wp := workerpool.New(c.Int("worker"))
	bar := progressbar.NewOptions(int(endBlkNum-startBlkNum+1),
		progressbar.OptionSetDescription("Verifying..."),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
	)
	for i := startBlkNum; i <= endBlkNum; i++ {
		i := i
		wp.Submit(func() {
			defer bar.Add(1)
			if err := verifyAction(dao, db, i); errors.Cause(err) == ErrActionNoMatch {
				fmt.Printf("\nblkNum=%d action no match \n", i)
			} else if err != nil {
				fmt.Printf("\nblkNum=%d action verify error=%v \n", i, err)
			}
			if err := verifyReceipt(dao, db, i); errors.Cause(err) == ErrReceiptNoMatch {
				fmt.Printf("\nblkNum=%d receipt no match \n", i)
			} else if err != nil {
				fmt.Printf("\nblkNum=%d receipt verify error=%v \n", i, err)
			}
			if err := verifyTransactions(dao, db, i); errors.Cause(err) == ErrTransactionNoMatch {
				fmt.Printf("\nblkNum=%d %v \n", i, err)
			} else if err != nil {
				fmt.Printf("\nblkNum=%d transaction verify error=%v \n", i, err)
			}

		})
	}
	wp.StopWait()
	return nil
}

func verifyAction(dao blockdao.BlockDAO, db *gorm.DB, height uint64) error {
	blk, err := dao.GetBlockByHeight(height)
	if err != nil {
		return err
	}

	var block models.Block
	if err := db.Table("block").Where("block_height = ?", height).Find(&block).Error; err != nil {
		return err
	}
	blkHash := blk.HashBlock()
	if block.BlockHash != hex.EncodeToString(blkHash[:]) &&
		block.NumActions != len(blk.Actions) {
		return ErrActionNoMatch
	}

	var actions []models.BlockAction
	if err := db.Table("block_action").Where("block_height = ?", height).Find(&actions).Error; err != nil {
		return err
	}
	if len(actions) != len(blk.Actions) {
		return errors.Wrapf(ErrActionNoMatch, "blkNum=%d action len not match", height)
	}
	for _, selp := range blk.Actions {
		actionHash, _ := selp.Hash()
		gasPrice := decimal.NewFromBigInt(selp.GasPrice(), 0)
		gasLimit := selp.GasLimit()
		nonce := selp.Nonce()

		act := selp.Action()
		actionType := getActionTypeString(act)
		amount, payload := getPayloadAmount(act)
		sender, _ := address.FromBytes(selp.SrcPubkey().Hash())
		dst, _ := selp.Destination()

		for i, action := range actions {
			if hex.EncodeToString(actionHash[:]) == action.ActionHash &&
				gasPrice.Equal(action.GasPrice) &&
				gasLimit == action.GasLimit &&
				nonce == action.Nonce &&
				actionType == action.ActionType &&
				action.Amount.Equal(decimal.NewFromBigInt(amount, 0)) &&
				bytes.Equal(payload, action.Payload) &&
				sender.String() == action.Sender &&
				dst == action.Recipient {
				actions = append(actions[:i], actions[i+1:]...)
				break
			}
		}
		if len(actions) == 0 {
			break
		}
	}
	if len(actions) != 0 {
		return errors.Wrapf(ErrActionNoMatch, "blkNum=%d action not match", height)
	}
	return nil
}

func verifyReceipt(dao blockdao.BlockDAO, db *gorm.DB, height uint64) error {
	daoReceipts, err := dao.GetReceipts(height)
	if err != nil {
		return err
	}
	var receipts []models.BlockReceipt
	if err := db.Table("block_receipts").Where("block_height = ?", height).Find(&receipts).Error; err != nil {
		return err
	}
	if len(daoReceipts) != len(receipts) {
		return errors.Wrapf(ErrReceiptNoMatch, "blkNum=%d receipt len not match", height)
	}
	for _, receipt := range daoReceipts {
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		for i, dbReceipt := range receipts {
			if actionHash == dbReceipt.ActionHash &&
				receipt.Status == dbReceipt.Status &&
				receipt.GasConsumed == dbReceipt.GasConsumed &&
				receipt.ContractAddress == dbReceipt.ContractAddress &&
				receipt.ExecutionRevertMsg() == dbReceipt.ExecutionRevertMsg {
				receipts = append(receipts[:i], receipts[i+1:]...)
				break
			}
		}
		if len(receipts) == 0 {
			break
		}
	}
	if len(receipts) != 0 {
		return errors.Wrapf(ErrReceiptNoMatch, "blkNum=%d receipt not match", height)
	}

	var logs []models.BlockReceiptLog
	if err := db.Table("block_receipt_logs").Where("block_height = ?", height).Find(&logs).Error; err != nil {
		return err
	}
	for _, receipt := range daoReceipts {
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		for _, log := range receipt.Logs() {
			for i, dbLog := range logs {
				topic0, topic1, topic2, topic3 := parseTopics(log.Topics)
				if actionHash == dbLog.ActionHash &&
					log.Address == dbLog.Address &&
					topic0 == dbLog.Topic0 &&
					topic1 == dbLog.Topic1 &&
					topic2 == dbLog.Topic2 &&
					topic3 == dbLog.Topic3 &&
					bytes.Equal(log.Data, dbLog.Data) {
					logs = append(logs[:i], logs[i+1:]...)
					break
				}
			}
		}
	}
	if len(logs) != 0 {
		return errors.Wrapf(ErrReceiptNoMatch, "blkNum=%d receipt log not match", height)
	}
	return nil
}

func verifyTransactions(dao blockdao.BlockDAO, db *gorm.DB, height uint64) error {
	daoTransactions, err := dao.TransactionLogs(height)
	if err != nil {
		return err
	}
	var transactions []models.BlockReceiptTransaction
	if err := db.Table("block_receipt_transactions").Where("block_height = ?", height).Find(&transactions).Error; err != nil {
		return err
	}
	daoLen := 0
	for _, transactionLogs := range daoTransactions.GetLogs() {
		daoLen += len(transactionLogs.GetTransactions())
	}
	if daoLen != len(transactions) {
		return errors.Wrapf(ErrTransactionNoMatch, "blkNum=%d transaction len not match, dao[%d]!=db[%d]", height, daoLen, len(transactions))
	}
	for _, transactionLogs := range daoTransactions.GetLogs() {
		actionHash := hex.EncodeToString(transactionLogs.ActionHash[:])
		for _, transaction := range transactionLogs.GetTransactions() {
			actType := getTransactionType(transaction.Type)
			ok := false
			for i, dbTransaction := range transactions {
				if actionHash == dbTransaction.ActionHash &&
					transaction.Sender == dbTransaction.Sender &&
					transaction.Recipient == dbTransaction.Recipient &&
					dbTransaction.Amount.String() == transaction.Amount &&
					actType == dbTransaction.Type {
					transactions = append(transactions[:i], transactions[i+1:]...)
					ok = true
					break
				}
			}
			if !ok {
				return errors.Wrapf(ErrTransactionNoMatch, "blkNum=%d transaction not match", height)
			}
		}
	}

	return nil
}

func firstLowerCase(s string) string {
	if len(s) == 0 {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func getActionTypeString(action action.Action) string {
	actionType := fmt.Sprintf("%T", action)
	return firstLowerCase(strings.TrimLeft(actionType, "*action."))
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

func getTransactionType(t iotextypes.TransactionLogType) string {
	switch {
	case t == iotextypes.TransactionLogType_IN_CONTRACT_TRANSFER:
		return execution
	case t == iotextypes.TransactionLogType_WITHDRAW_BUCKET:
		return stakeWithdraw
	case t == iotextypes.TransactionLogType_CREATE_BUCKET:
		return stakeCreate
	case t == iotextypes.TransactionLogType_DEPOSIT_TO_BUCKET:
		return stakeAddDeposit
	case t == iotextypes.TransactionLogType_CLAIM_FROM_REWARDING_FUND:
		return claimFromRewardingFund
	case t == iotextypes.TransactionLogType_DEPOSIT_TO_REWARDING_FUND:
		return depositToRewardingFund
	case t == iotextypes.TransactionLogType_CANDIDATE_REGISTRATION_FEE:
		return candidateRegisterFee
	case t == iotextypes.TransactionLogType_CANDIDATE_SELF_STAKE:
		return candidateRegisterSelfStake
	case t == iotextypes.TransactionLogType_GAS_FEE:
		return gasFee
	case t == iotextypes.TransactionLogType_NATIVE_TRANSFER:
		return transfer
	}
	return ""
}

func parseTopics(topics []hash.Hash256) (topic0, topic1, topic2, topic3 string) {
	if len(topics) > 3 {
		topic3 = hex.EncodeToString(topics[3][:])
	}
	if len(topics) > 2 {
		topic2 = hex.EncodeToString(topics[2][:])
	}
	if len(topics) > 1 {
		topic1 = hex.EncodeToString(topics[1][:])
	}
	if len(topics) > 0 {
		topic0 = hex.EncodeToString(topics[0][:])
	}
	return
}
