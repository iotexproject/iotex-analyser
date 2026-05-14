package staking_actions

import (
	"context"
	"encoding/hex"
	stderrors "errors"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const VERSION = "2.2.0"

var (
	// topic hashes for exit queue events, computed from ABI event signatures
	topicDeactivationRequested = hash.Hash256(gethcrypto.Keccak256Hash([]byte("CandidateDeactivationRequested(address)")))
	topicDeactivationScheduled = hash.Hash256(gethcrypto.Keccak256Hash([]byte("CandidateDeactivationScheduled(address,uint64)")))
	topicDeactivated           = hash.Hash256(gethcrypto.Keccak256Hash([]byte("CandidateDeactivated(address)")))
	// topic hashes for Solidity-style staking events that carry the bucket
	// index in log.data (first 32 bytes), not in topic[1] like the legacy
	// receipt-log format. Emitted by candidateRegisterWithBLS and friends
	// once the chain stops emitting the legacy non-postFairbankMigration log.
	topicStaked             = hash.Hash256(gethcrypto.Keccak256Hash([]byte("Staked(address,address,uint64,uint256,uint32,bool)")))
	topicCandidateActivated = hash.Hash256(gethcrypto.Keccak256Hash([]byte("CandidateActivated(address,uint64)")))
)

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

type StakingActionPlugin struct {
	plugin.PluginShadow
}

func (b StakingActionPlugin) Name() string {
	return b.ShadowName("staking_actions")
}

func (b StakingActionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b StakingActionPlugin) DependentPlugins() []string {
	return []string{"candidate", "slash"}
}

func (b StakingActionPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), b.ShadowTable(&models.StakingActions{}), &models.CandidateExitQueue{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	// CandidateExitQueue was added after staking_actions was first deployed;
	// on existing installs AutoMigrate is a no-op (height > 0) so create it
	// explicitly if missing.
	if err := db.EnsureTables(&models.CandidateExitQueue{}); err != nil {
		return errors.Wrapf(err, "failed to ensure candidate_exit_queue table for plugin %s", b.Name())
	}

	var ok bool
	unSelfStake, ok = new(big.Int).SetString("000000000000000000000000000000000000000000000000ffffffffffffffff", 16)
	if !ok {
		return errors.New("can not convert string to bigint with plugin %s:" + b.Name())
	}

	return nil
}

func (b StakingActionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {

	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if err := b.handleBlock(ctx, blk, tx); err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return errors.Wrap(err, "")
}

func (b StakingActionPlugin) BatchSize() int {
	return 0
}

func (b StakingActionPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, blk := range blks {
			if err := b.handleBlock(ctx, blk, tx); err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blks[len(blks)-1].Height())
	})
	return err
}

func (b StakingActionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b StakingActionPlugin) Version() string {
	return VERSION
}

func (b StakingActionPlugin) handleBlock(ctx context.Context, blk *block.Block, tx *gorm.DB) error {
	var stakingAction models.StakingActions
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
			if log.Address != StakingProtocolAddress || len(log.Topics) == 0 {
				continue
			}
			// Solidity-style events (post-Fairbank/Yap, emitted by
			// candidateRegisterWithBLS etc.): bucket index lives in the
			// first 32 bytes of log.data — topic[1] is the indexed
			// candidate/voter address instead.
			switch log.Topics[0] {
			case topicStaked, topicCandidateActivated:
				if len(log.Data) >= 32 {
					bucketIndex := new(big.Int).SetBytes(log.Data[:32])
					if bucketID, ok := validBucketIndex(bucketIndex); ok {
						bucketMap[actHash] = bucketID
					}
				}
				continue
			}
			// Legacy receipt-log format (newReceiptLog without events):
			// topic[1] is the bucket index. Kept for backward compatibility
			// with blocks indexed before the Solidity-events path.
			if len(log.Topics) > 1 {
				bucketIndex := new(big.Int).SetBytes(log.Topics[1][:])
				bucketID, ok := validBucketIndex(bucketIndex)
				if !ok {
					continue
				}
				bucketMap[actHash] = bucketID
			}
		}
		switch a := act.(type) {
		case *action.CreateStake:
			cadidateAddr, err := getCandidateAddressByName(a.Candidate(), blk.Height())
			if err != nil {
				return err
			}

			bucketID, ok := bucketMap[actHash]
			if !ok {
				return errors.New("can not found bucketID with actHash:" + actHash)
			}
			stakingAction = models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     bucketID,
				Sender:       sender.String(),
				OwnerAddress: sender.String(),
				ActHash:      actHash,
				Candidate:    cadidateAddr,
				Amount:       decimal.NewFromBigInt(a.Amount(), 0),
				ActType:      "StakeCreate",
				AutoStake:    a.AutoStake(),
				Duration:     a.Duration(),
			}
			if err := tx.Create(b.ShadowTable(&stakingAction)).Error; err != nil {
				return err
			}
		case *action.TransferStake:
			bucketID := a.BucketIndex()
			decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketID)
			if err != nil {
				return errors.Wrapf(err, errBucketSumAmount, bucketID)
			}
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			stakingAction = models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     bucketID,
				OwnerAddress: info.OwnerAddress,
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "TransferStake",
				Duration:     info.Duration,
				Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
			}
			if err := tx.Create(&stakingAction).Error; err != nil {
				return err
			}
			stakingAction = models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     bucketID,
				OwnerAddress: a.VoterAddress().String(),
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "TransferStake",
				Duration:     info.Duration,
				Amount:       decmailAmount,
			}
			if err := tx.Create(b.ShadowTable(&stakingAction)).Error; err != nil {
				return err
			}
		case *action.Restake:
			bucketID := a.BucketIndex()
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			// fix greenland (height=6544441) restake
			fixAmount := decimal.NewFromInt(0)
			if blk.Height() < genesis.Default.GreenlandBlockHeight {
				fixAmount, err = b.getFixBucketSumAmountByBucketID(tx, bucketID)
				if err != nil {
					return errors.Wrapf(err, errBucketSumAmount, bucketID)
				}
			}
			stakingAction = models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     bucketID,
				OwnerAddress: sender.String(),
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    a.AutoStake(),
				ActType:      "Restake",
				Duration:     a.Duration(),
				Amount:       fixAmount,
			}
			if err := tx.Create(b.ShadowTable(&stakingAction)).Error; err != nil {
				return err
			}
		case *action.ChangeCandidate:
			bucketID := a.BucketIndex()
			decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketID)
			if err != nil {
				return errors.Wrapf(err, errBucketSumAmount, bucketID)
			}
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			stakingAction = models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     bucketID,
				OwnerAddress: info.OwnerAddress,
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "ChangeCandidate",
				Duration:     info.Duration,
				Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
			}
			if err := tx.Create(b.ShadowTable(&stakingAction)).Error; err != nil {
				return err
			}
			cadidateAddr, err := getCandidateAddressByName(a.Candidate(), blk.Height())
			if err != nil {
				return err
			}
			stakingAction = models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     bucketID,
				OwnerAddress: info.OwnerAddress,
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    cadidateAddr,
				AutoStake:    info.AutoStake,
				ActType:      "ChangeCandidate",
				Duration:     info.Duration,
				Amount:       decmailAmount,
			}
			if err := tx.Create(b.ShadowTable(&stakingAction)).Error; err != nil {
				return err
			}
			//TODO: candidate update has no bucketID
		case *action.CandidateUpdate:
			continue
		case *action.DepositToStake:
			bucketID := a.BucketIndex()
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			stakingAction = models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     bucketID,
				OwnerAddress: info.OwnerAddress,
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "DepositToStake",
				Duration:     info.Duration,
				Amount:       decimal.NewFromBigInt(a.Amount(), 0),
			}
			if err := tx.Create(b.ShadowTable(&stakingAction)).Error; err != nil {
				return err
			}
		case *action.Unstake:
			bucketID := a.BucketIndex()
			decmailAmount, err := b.getBucketSumAmountByBucketID(tx, bucketID)
			if err != nil {
				return errors.Wrap(err, "getBucketSumAmountByBucketID error")
			}
			info, err := b.getBucketInfoAddressByBucketID(tx, bucketID)
			if err != nil {
				return errors.Wrap(err, errBucketInfoAddressByBucketID)
			}
			stakingAction = models.StakingActions{
				BlockHeight:  blk.Height(),
				BucketID:     bucketID,
				OwnerAddress: sender.String(),
				Sender:       sender.String(),
				ActHash:      actHash,
				Candidate:    info.Candidate,
				AutoStake:    info.AutoStake,
				ActType:      "Unstake",
				Duration:     info.Duration,
				Amount:       decmailAmount.Mul(decimal.NewFromInt(-1)),
			}
			if err := tx.Create(b.ShadowTable(&stakingAction)).Error; err != nil {
				return err
			}
		case *action.CandidateRegister:
			bucketID, ok := bucketMap[actHash]
			if !ok {
				return errors.New("can not found bucketID with actHash:" + actHash)
			}

			if bucketID != unSelfStake.Uint64() {
				stakingAction = models.StakingActions{
					BlockHeight:  blk.Height(),
					BucketID:     bucketID,
					Sender:       sender.String(),
					OwnerAddress: a.OwnerAddress().String(),
					ActHash:      actHash,
					Candidate:    a.OwnerAddress().String(),
					Amount:       decimal.NewFromBigInt(a.Amount(), 0),
					ActType:      "CandidateRegister",
					AutoStake:    a.AutoStake(),
					Duration:     a.Duration(),
				}
				if err := tx.Create(b.ShadowTable(&stakingAction)).Error; err != nil {
					return err
				}
			}
		case *action.GrantReward:
			if a.RewardType() != action.EpochReward {
				continue
			}
			slashs, err := models.FetchSlashByActionHash(actHash, tx)
			if err != nil {
				return errors.Wrapf(err, "FetchSlashByActionHash error, actionHash: %s", actHash)
			}
			for _, slash := range slashs {
				info, err := b.getBucketInfoAddressByBucketID(tx, slash.BucketID)
				if err != nil {
					return errors.Wrap(err, errBucketInfoAddressByBucketID)
				}
				stakingAction = models.StakingActions{
					BlockHeight:  blk.Height(),
					BucketID:     slash.BucketID,
					OwnerAddress: info.OwnerAddress,
					Sender:       sender.String(),
					ActHash:      actHash,
					Candidate:    slash.CandidateID,
					ActType:      "SlashCandidate",
					Amount:       slash.Amount.Neg(),
					AutoStake:    info.AutoStake,
					Duration:     info.Duration,
				}
				if err := tx.Create(b.ShadowTable(&stakingAction)).Error; err != nil {
					return err
				}
			}
		case *action.CandidateDeactivate:
			candidateIdentity, err := candidateIdentityFromLogs(receipt.Logs(), topicDeactivationRequested, topicDeactivated)
			if err != nil {
				return errors.Wrapf(err, "failed to get candidate identity for CandidateDeactivate, actHash: %s", actHash)
			}
			cand := &models.Candidate{}
			if err := cand.FetchByCandidateIDWithHeight(candidateIdentity, blk.Height(), tx); err != nil {
				return errors.Wrapf(err, "failed to fetch candidate by identity %s", candidateIdentity)
			}
			actType := "CandidateDeactivateRequest"
			if a.Op() == action.CandidateDeactivateOpConfirm {
				actType = "CandidateDeactivateConfirm"
			}
			stakingAction = models.StakingActions{
				BlockHeight:  blk.Height(),
				Sender:       sender.String(),
				OwnerAddress: cand.OwnerAddress,
				ActHash:      actHash,
				Candidate:    candidateIdentity,
				ActType:      actType,
			}
			if err := tx.Create(b.ShadowTable(&stakingAction)).Error; err != nil {
				return err
			}
			if err := b.updateExitQueue(tx, actType, blk.Height(), actHash, cand, 0); err != nil {
				return errors.Wrapf(err, "failed to update exit queue for %s, actHash: %s", actType, actHash)
			}
		case *action.ScheduleCandidateDeactivation:
			candidateIdentity := a.Delegate().String()
			scheduledAt, err := scheduledAtFromLogsOrChain(receipt.Logs(), candidateIdentity, blk.Height())
			if err != nil {
				return errors.Wrapf(err, "failed to get scheduledAt for ScheduleCandidateDeactivation, actHash: %s", actHash)
			}
			cand := &models.Candidate{}
			if err := cand.FetchByCandidateIDWithHeight(candidateIdentity, blk.Height(), tx); err != nil {
				return errors.Wrapf(err, "failed to fetch candidate by identity %s", candidateIdentity)
			}
			stakingAction = models.StakingActions{
				BlockHeight:  blk.Height(),
				Sender:       sender.String(),
				OwnerAddress: cand.OwnerAddress,
				ActHash:      actHash,
				Candidate:    candidateIdentity,
				ActType:      "ScheduleCandidateDeactivation",
			}
			if err := tx.Create(b.ShadowTable(&stakingAction)).Error; err != nil {
				return err
			}
			if err := b.updateExitQueue(tx, "ScheduleCandidateDeactivation", blk.Height(), actHash, cand, scheduledAt); err != nil {
				return errors.Wrapf(err, "failed to update exit queue for ScheduleCandidateDeactivation, actHash: %s", actHash)
			}
		}
	}
	return nil
}

func validBucketIndex(bucketIndex *big.Int) (uint64, bool) {
	if bucketIndex == nil || bucketIndex.Sign() < 0 || bucketIndex.BitLen() > 64 {
		return 0, false
	}
	bucketID := bucketIndex.Uint64()
	if bucketID == ^uint64(0) || bucketIndex.IsInt64() {
		return bucketID, true
	}
	return 0, false
}

// candidateIdentityFromLogs extracts the candidate IoTeX address from staking protocol receipt logs.
// It matches the first log whose topic[0] is one of the provided event topics.
func candidateIdentityFromLogs(logs []*action.Log, topics ...hash.Hash256) (string, error) {
	for _, log := range logs {
		if log.Address != StakingProtocolAddress || len(log.Topics) < 2 {
			continue
		}
		for _, topic := range topics {
			if log.Topics[0] == topic {
				addr, err := iotexAddrFromTopic(log.Topics[1])
				if err != nil {
					return "", err
				}
				return addr.String(), nil
			}
		}
	}
	return "", errors.New("candidate identity not found in receipt logs")
}

// errLogDataTooShort signals that the schedule event was found but its data
// payload is missing/short. iotex-core <=v2.4.0 has a bug in receipt_log.go
// Build() that drops r.data on the postFairbankMigration path; only the schedule
// event exposes it (the only deactivation event with non-indexed inputs).
var errLogDataTooShort = errors.New("CandidateDeactivationScheduled log data too short")

// scheduledAtFromLogs decodes the scheduled block height from a
// CandidateDeactivationScheduled event log. Returns errLogDataTooShort if the
// event is present but its data payload is missing — callers that have access
// to a chain client should fall back to chain state via scheduledAtFromLogsOrChain.
func scheduledAtFromLogs(logs []*action.Log) (uint64, error) {
	for _, log := range logs {
		if log.Address != StakingProtocolAddress || len(log.Topics) < 2 {
			continue
		}
		if log.Topics[0] == topicDeactivationScheduled {
			if len(log.Data) < 32 {
				return 0, errLogDataTooShort
			}
			return new(big.Int).SetBytes(log.Data[len(log.Data)-8:]).Uint64(), nil
		}
	}
	return 0, errors.New("CandidateDeactivationScheduled event not found in receipt logs")
}

// scheduledAtFromLogsOrChain wraps scheduledAtFromLogs with a fallback that
// queries chain state at the schedule block when log.Data was dropped by the
// chain bug. Confirm in iotex-core does NOT reset DeactivatedAt, so the
// historical read at the schedule block always carries the correct value.
func scheduledAtFromLogsOrChain(logs []*action.Log, candidateIdentity string, blockHeight uint64) (uint64, error) {
	v, err := scheduledAtFromLogs(logs)
	if err == nil {
		return v, nil
	}
	if !errors.Is(err, errLogDataTooShort) {
		return 0, err
	}
	v, fallbackErr := readScheduledAtFromChain(candidateIdentity, blockHeight)
	if fallbackErr != nil {
		// Propagate the failure rather than persisting an incorrect 0 — without
		// scheduledAt the eligible-from column is meaningless, and downstream
		// consumers can't tell "unknown" from a real 0 once it's in the DB.
		// Halting the indexer here lets the operator fix the chain client and
		// reindex this block instead of getting silent data corruption.
		return 0, errors.Wrapf(fallbackErr,
			"scheduledAt fallback ReadState failed for candidate %s at height %d",
			candidateIdentity, blockHeight)
	}
	return v, nil
}

// readScheduledAtFromChain calls ReadState(CANDIDATE_BY_ADDRESS, Height=blockHeight) and
// returns the candidate's DeactivatedAt at that height.
func readScheduledAtFromChain(candidateIdentity string, blockHeight uint64) (uint64, error) {
	cli := kernel.ChainClient()
	if cli == nil {
		return 0, errors.New("chain client unavailable")
	}
	methodBytes, err := proto.Marshal(&iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_CANDIDATE_BY_ADDRESS,
	})
	if err != nil {
		return 0, err
	}
	reqBytes, err := proto.Marshal(&iotexapi.ReadStakingDataRequest{
		Request: &iotexapi.ReadStakingDataRequest_CandidateByAddress_{
			CandidateByAddress: &iotexapi.ReadStakingDataRequest_CandidateByAddress{
				OwnerAddr: candidateIdentity,
				Id:        candidateIdentity,
			},
		},
	})
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := cli.ReadState(ctx, &iotexapi.ReadStateRequest{
		ProtocolID: []byte("staking"),
		MethodName: methodBytes,
		Arguments:  [][]byte{reqBytes},
		Height:     strconv.FormatUint(blockHeight, 10),
	})
	if err != nil {
		return 0, errors.Wrap(err, "ReadState(CANDIDATE_BY_ADDRESS) failed")
	}
	var c iotextypes.CandidateV2
	if err := proto.Unmarshal(resp.Data, &c); err != nil {
		return 0, errors.Wrap(err, "unmarshal CandidateV2")
	}
	if c.OwnerAddress == "" && c.Id == "" {
		return 0, errors.Errorf("candidate %q not found at height %d", candidateIdentity, blockHeight)
	}
	return c.DeactivatedAt, nil
}

// iotexAddrFromTopic converts a 32-byte topic (right-padded ETH address) to an IoTeX address.
func iotexAddrFromTopic(topic hash.Hash256) (address.Address, error) {
	ethAddr := common.BytesToAddress(topic[12:])
	return address.FromBytes(ethAddr.Bytes())
}

// updateExitQueue upserts the candidate_exit_queue row based on the action type.
//
// Schedule/Confirm look up the latest matching row by id DESC and update by
// primary key, rather than UPDATE WHERE (candidate_identity, status). This
// pins each Schedule/Confirm to exactly one preceding row even if multiple
// historical rows for the same candidate exist (e.g. after a previous full
// exit cycle plus reactivation, or stale `requested`/`scheduled` rows left
// behind by missed indexing).
func (b StakingActionPlugin) updateExitQueue(tx *gorm.DB, actType string, height uint64, actHash string, cand *models.Candidate, scheduledAt uint64) error {
	switch actType {
	case "CandidateDeactivateRequest":
		row := &models.CandidateExitQueue{
			CandidateName:     cand.Name,
			CandidateIdentity: cand.CandidateID,
			Status:            "requested",
			RequestHeight:     height,
			RequestHash:       actHash,
		}
		return tx.Create(row).Error
	case "ScheduleCandidateDeactivation":
		var row models.CandidateExitQueue
		err := tx.Where("candidate_identity = ? AND status = ?", cand.CandidateID, "requested").
			Order("id DESC").
			First(&row).Error
		switch {
		case stderrors.Is(err, gorm.ErrRecordNotFound):
			return errors.Errorf("no 'requested' row to schedule for candidate %s", cand.CandidateID)
		case err != nil:
			return errors.Wrapf(err, "failed to look up 'requested' row for candidate %s", cand.CandidateID)
		}
		return tx.Model(&models.CandidateExitQueue{}).
			Where("id = ?", row.ID).
			Updates(map[string]interface{}{
				"status":          "scheduled",
				"schedule_height": height,
				"schedule_hash":   actHash,
				"scheduled_at":    scheduledAt,
			}).Error
	case "CandidateDeactivateConfirm":
		var row models.CandidateExitQueue
		err := tx.Where("candidate_identity = ? AND status = ?", cand.CandidateID, "scheduled").
			Order("id DESC").
			First(&row).Error
		switch {
		case stderrors.Is(err, gorm.ErrRecordNotFound):
			return errors.Errorf("no 'scheduled' row to confirm for candidate %s", cand.CandidateID)
		case err != nil:
			return errors.Wrapf(err, "failed to look up 'scheduled' row for candidate %s", cand.CandidateID)
		}
		return tx.Model(&models.CandidateExitQueue{}).
			Where("id = ?", row.ID).
			Updates(map[string]interface{}{
				"status":         "confirmed",
				"confirm_height": height,
				"confirm_hash":   actHash,
			}).Error
	}
	return nil
}
