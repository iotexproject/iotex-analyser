package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"unicode"

	"github.com/google/go-cmp/cmp"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
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
	// ErrActionNoMatch means action no match
	ErrActionNoMatch = errors.New("action no match")
	// ErrReceiptNoMatch means receipt no match
	ErrReceiptNoMatch = errors.New("receipt no match")
	// ErrTransactionNoMatch means transaction no match
	ErrTransactionNoMatch = errors.New("transaction no match")
)

type diffBlk struct {
	BlkHash         string
	ProducerAddress string
	Timestamp       int64
	NumActions      int
}

type diffAction struct {
	ActionHash string
	GasPrice   decimal.Decimal
	GasLimit   uint64
	Nonce      uint64
	ActionType string
	Amount     decimal.Decimal
	Payload    []byte
	Sender     string
	Recipient  string
}

type diffActions []diffAction

func (d diffActions) Len() int {
	return len(d)
}
func (d diffActions) Less(i, j int) bool {
	return d[i].ActionHash < d[j].ActionHash
}
func (d diffActions) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func verifyAction(blk *block.Block, db *gorm.DB, height uint64) error {
	var block models.Block
	if err := db.Table("block").Where("block_height = ?", height).Find(&block).Error; err != nil {
		return err
	}
	blkHash := blk.HashBlock()
	got, want := diffBlk{
		BlkHash:         block.BlockHash,
		ProducerAddress: block.ProducerAddress,
		Timestamp:       block.Timestamp.Unix(),
		NumActions:      block.NumActions,
	}, diffBlk{
		BlkHash:         hex.EncodeToString(blkHash[:]),
		ProducerAddress: blk.ProducerAddress(),
		Timestamp:       blk.Timestamp().Unix(),
		NumActions:      len(blk.Actions),
	}
	if diff := cmp.Diff(want, got); diff != "" {
		fmt.Printf("\nmismatch blk (-want +got):\n%s\n", diff)
		return errors.Wrapf(ErrActionNoMatch, "mismatch blk")
	}

	var actions []models.BlockAction
	if err := db.Table("block_action").Where("block_height = ?", height).Find(&actions).Error; err != nil {
		return err
	}
	if len(actions) != len(blk.Actions) {
		return errors.Wrapf(ErrActionNoMatch, "blkNum=%d action len not match", height)
	}
	var want1, got1 diffActions
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
		want1 = append(want1, diffAction{
			ActionHash: hex.EncodeToString(actionHash[:]),
			GasPrice:   gasPrice,
			GasLimit:   gasLimit,
			Nonce:      nonce,
			ActionType: actionType,
			Amount:     decimal.NewFromBigInt(amount, 0),
			Payload:    payload,
			Sender:     sender.String(),
			Recipient:  strings.ToLower(dst),
		})
	}

	for _, action := range actions {
		got1 = append(got1, diffAction{
			ActionHash: action.ActionHash,
			GasPrice:   action.GasPrice,
			GasLimit:   action.GasLimit,
			Nonce:      action.Nonce,
			ActionType: action.ActionType,
			Amount:     action.Amount,
			Payload:    action.Payload,
			Sender:     action.Sender,
			Recipient:  action.Recipient,
		})
	}

	sort.Sort(want1)
	sort.Sort(got1)
	if diff := cmp.Diff(want1, got1); diff != "" {
		fmt.Printf("\nmismatch actions (-want +got):\n%s\n", diff)
		return errors.Wrapf(ErrActionNoMatch, "mismatch actions")
	}
	return nil
}

type diffReceipt struct {
	ActionHash         string
	Status             uint64
	GasConsumed        uint64
	ContractAddress    string
	ExecutionRevertMsg string
}
type diffReceipts []diffReceipt

func (d diffReceipts) Len() int {
	return len(d)
}
func (d diffReceipts) Less(i, j int) bool {
	return d[i].ActionHash < d[j].ActionHash
}
func (d diffReceipts) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

type diffReceiptLog struct {
	ActionHash string
	Address    string
	Topic0     string
	Topic1     string
	Topic2     string
	Topic3     string
	Data       []byte
}
type diffReceiptLogs []diffReceiptLog

func (d diffReceiptLogs) Len() int {
	return len(d)
}
func (d diffReceiptLogs) Less(i, j int) bool {
	return d[i].ActionHash < d[j].ActionHash
}
func (d diffReceiptLogs) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
func verifyReceipt(blk *block.Block, db *gorm.DB, height uint64) error {
	daoReceipts := blk.Receipts
	var receipts []models.BlockReceipt
	if err := db.Table("block_receipts").Where("block_height = ?", height).Find(&receipts).Error; err != nil {
		return err
	}
	if len(daoReceipts) != len(receipts) {
		return errors.Wrapf(ErrReceiptNoMatch, "blkNum=%d receipt len not match", height)
	}
	var want, got diffReceipts
	for _, receipt := range daoReceipts {
		want = append(want, diffReceipt{
			ActionHash:         hex.EncodeToString(receipt.ActionHash[:]),
			Status:             receipt.Status,
			GasConsumed:        receipt.GasConsumed,
			ContractAddress:    receipt.ContractAddress,
			ExecutionRevertMsg: receipt.ExecutionRevertMsg(),
		})
	}
	for _, receipt := range receipts {
		got = append(got, diffReceipt{
			ActionHash:         receipt.ActionHash,
			Status:             receipt.Status,
			GasConsumed:        receipt.GasConsumed,
			ContractAddress:    receipt.ContractAddress,
			ExecutionRevertMsg: receipt.ExecutionRevertMsg,
		})
	}
	sort.Sort(want)
	sort.Sort(got)
	if diff := cmp.Diff(want, got); diff != "" {
		fmt.Printf("\nmismatch receipt (-want +got):\n%s\n", diff)
		return errors.Wrapf(ErrReceiptNoMatch, "mismatch receipt")
	}

	var logs []models.BlockReceiptLog
	if err := db.Table("block_receipt_logs").Where("block_height = ?", height).Find(&logs).Error; err != nil {
		return err
	}
	var want1, got1 diffReceiptLogs
	for _, receipt := range daoReceipts {
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		for _, log := range receipt.Logs() {
			topic0, topic1, topic2, topic3 := parseTopics(log.Topics)
			logData := log.Data
			if log.Data == nil {
				logData = []byte("")
			}
			want1 = append(want1, diffReceiptLog{
				ActionHash: actionHash,
				Address:    log.Address,
				Topic0:     topic0,
				Topic1:     topic1,
				Topic2:     topic2,
				Topic3:     topic3,
				Data:       logData,
			})

		}
	}
	for _, dbLog := range logs {
		got1 = append(got1, diffReceiptLog{
			ActionHash: dbLog.ActionHash,
			Address:    dbLog.Address,
			Topic0:     dbLog.Topic0,
			Topic1:     dbLog.Topic1,
			Topic2:     dbLog.Topic2,
			Topic3:     dbLog.Topic3,
			Data:       dbLog.Data,
		})
	}
	sort.Sort(want1)
	sort.Sort(got1)
	if diff := cmp.Diff(want1, got1); diff != "" {
		fmt.Printf("\nmismatch receipt logs (-want +got):\n%s\n", diff)
		return errors.Wrapf(ErrReceiptNoMatch, "mismatch receipt logs")
	}
	return nil
}

type diffTransaction struct {
	ActionHash string
	Sender     string
	Recipient  string
	Amount     string
	ActionType string
}
type diffTransactions []diffTransaction

func (d diffTransactions) Len() int {
	return len(d)
}
func (d diffTransactions) Less(i, j int) bool {
	return d[i].ActionHash < d[j].ActionHash
}
func (d diffTransactions) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func verifyTransactions(blk *block.Block, db *gorm.DB, height uint64) error {
	daoReceipts := blk.Receipts
	var transactions []models.BlockReceiptTransaction
	if err := db.Table("block_receipt_transactions").Where("block_height = ?", height).Find(&transactions).Error; err != nil {
		return err
	}
	var want, got diffTransactions
	for _, receipt := range daoReceipts {
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		for _, transaction := range receipt.TransactionLogs() {
			actType := getTransactionType(transaction.Type)
			want = append(want, diffTransaction{
				ActionHash: actionHash,
				Sender:     transaction.Sender,
				Recipient:  transaction.Recipient,
				Amount:     transaction.Amount.String(),
				ActionType: actType,
			})
		}
	}
	for _, dbTransaction := range transactions {
		got = append(got, diffTransaction{
			ActionHash: dbTransaction.ActionHash,
			Sender:     dbTransaction.Sender,
			Recipient:  dbTransaction.Recipient,
			Amount:     dbTransaction.Amount.String(),
			ActionType: dbTransaction.Type,
		})
	}
	sort.Sort(want)
	sort.Sort(got)
	if diff := cmp.Diff(want, got); diff != "" {
		fmt.Printf("\nmismatch transactions (-want +got):\n%s\n", diff)
		return errors.Wrapf(ErrReceiptNoMatch, "mismatch transactions")
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
