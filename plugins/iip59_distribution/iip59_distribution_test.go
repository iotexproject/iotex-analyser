package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/iotexproject/go-pkgs/crypto"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	dbpkg "github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rewarding/distributedlog"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var testGormDB *gorm.DB

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().Port(9993),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("WARNING: embedded postgres failed to start: %v\n  DB tests will be skipped.\n", err)
		os.Exit(m.Run())
	}

	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=9993 sslmode=disable"
	var err error
	testGormDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		fmt.Printf("WARNING: failed to connect to embedded postgres: %v\n  DB tests will be skipped.\n", err)
		code := m.Run()
		pg.Stop()
		os.Exit(code)
	}

	dbpkg.SetDB(testGormDB)

	if err := testGormDB.AutoMigrate(
		&dbpkg.IndexHeight{}, &dbpkg.Store{},
		&models.IIP59DelegateDistribution{},
		&models.IIP59VoterReward{},
		&models.IIP59DelegateOptIn{},
		&models.IIP59VoterDestination{},
	); err != nil {
		fmt.Printf("WARNING: failed to migrate tables: %v\n  DB tests will be skipped.\n", err)
		testGormDB = nil
	}

	code := m.Run()
	pg.Stop()
	os.Exit(code)
}

func resetTestDB(t *testing.T) {
	t.Helper()
	if testGormDB == nil {
		t.Skip("embedded postgres not available")
	}
	for _, table := range []string{
		"iip59_delegate_distributions",
		"iip59_voter_rewards",
		"iip59_delegate_opt_ins",
		"iip59_voter_destinations",
	} {
		require.NoError(t, testGormDB.Exec("TRUNCATE TABLE "+table).Error)
	}
}

// distributedLog builds a DelegateDistributed receipt log from EventArgs, using
// the protocol's own encoder so the test cannot drift from what the chain emits.
func distributedLog(t *testing.T, args distributedlog.EventArgs) *action.Log {
	t.Helper()
	topics, data, err := distributedlog.Pack(args)
	require.NoError(t, err)
	return &action.Log{
		Address: address.RewardingProtocol,
		Topics:  topics,
		Data:    data,
	}
}

func blockWithLogs(t *testing.T, height uint64, logs ...*action.Log) *block.Block {
	t.Helper()
	var actHash hash.Hash256
	actHash[0] = byte(height)
	receipt := (&action.Receipt{
		Status:      uint64(iotextypes.ReceiptStatus_Success),
		BlockHeight: height,
		ActionHash:  actHash,
	}).AddLogs(logs...)

	tb := block.NewTestingBuilder()
	blk, err := tb.SetPrevBlockHash(hash.ZeroHash256).
		SetHeight(height).
		SetTimeStamp(time.Unix(int64(height), 0)).
		SetReceipts([]*action.Receipt{receipt}).
		SignAndBuild(identityset.PrivateKey(0))
	require.NoError(t, err)
	return &blk
}

func runBlock(t *testing.T, blk *block.Block) {
	t.Helper()
	require.NoError(t, testGormDB.Transaction(func(tx *gorm.DB) error {
		return handleBlock(blk, tx)
	}))
}

func chunkArgs(epoch uint64, delegate address.Address, snapshot hash.Hash256,
	commission *big.Int, voters []address.Address, amounts []*big.Int) distributedlog.EventArgs {
	pool := big.NewInt(0)
	for _, a := range amounts {
		pool.Add(pool, a)
	}
	return distributedlog.EventArgs{
		Epoch:             epoch,
		Delegate:          delegate,
		RewardAddr:        identityset.Address(2),
		EraCommission:     commission,
		ChunkVoterReward:  pool,
		SnapshotHash:      snapshot,
		Voters:            voters,
		Recipients:        voters,
		Amounts:           amounts,
		CompoundBucketIDs: make([]uint64, len(voters)),
		Compounded:        make([]bool, len(voters)),
	}
}

// TestChunkedSettlementAggregationSemantics is the test this plugin exists to
// get right.
//
// A settlement is drained across several blocks and each block emits its own
// DelegateDistributed log. The two amount fields carried by those logs
// aggregate in OPPOSITE directions, and nothing in their names says so:
//
//   - chunk_voter_reward is per-chunk (rewarding/voter_reward.go passes
//     rows.paid, accumulated within one block) and must be SUMMED.
//   - era_commission is an era constant read from the drain cursor and is
//     repeated verbatim in every chunk; SUMMING it multiplies the delegate's
//     commission by the number of blocks the drain took.
//
// The assertion below is what a downstream consumer would get wrong.
func TestChunkedSettlementAggregationSemantics(t *testing.T) {
	resetTestDB(t)
	r := require.New(t)

	delegate := identityset.Address(1)
	snapshot := hash.Hash256b([]byte("era-7-delegate-1"))
	commission := big.NewInt(500)

	// Same settlement, drained over two blocks: three voters, then two.
	runBlock(t, blockWithLogs(t, 100, distributedLog(t, chunkArgs(
		7, delegate, snapshot, commission,
		[]address.Address{identityset.Address(3), identityset.Address(4), identityset.Address(5)},
		[]*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)},
	))))
	runBlock(t, blockWithLogs(t, 101, distributedLog(t, chunkArgs(
		7, delegate, snapshot, commission,
		[]address.Address{identityset.Address(6), identityset.Address(7)},
		[]*big.Int{big.NewInt(40), big.NewInt(50)},
	))))

	var rows []models.IIP59DelegateDistribution
	r.NoError(testGormDB.Where("snapshot_hash = ?", hex.EncodeToString(snapshot[:])).
		Order("block_height").Find(&rows).Error)
	r.Len(rows, 2, "one row per chunk, not one per settlement")

	// era_commission: identical in both chunks. Summing gives 1000, which is
	// twice the real commission -- that is the bug this pins.
	r.Equal("500", rows[0].EraCommission.String())
	r.Equal("500", rows[1].EraCommission.String())

	// chunk_voter_reward: per-chunk, sums to the era total 150.
	r.Equal("60", rows[0].ChunkVoterReward.String())
	r.Equal("90", rows[1].ChunkVoterReward.String())

	var summed struct{ Total string }
	r.NoError(testGormDB.Raw(
		`SELECT COALESCE(SUM(chunk_voter_reward),0)::text AS total FROM iip59_delegate_distributions WHERE snapshot_hash = ?`,
		hex.EncodeToString(snapshot[:]),
	).Scan(&summed).Error)
	r.Equal("150", summed.Total)

	// And the per-voter rows must sum to the same figure -- the summary and the
	// fan-out are two views of one settlement and must not disagree.
	var payouts struct{ Total string }
	r.NoError(testGormDB.Raw(
		`SELECT COALESCE(SUM(amount),0)::text AS total FROM iip59_voter_rewards WHERE snapshot_hash = ?`,
		hex.EncodeToString(snapshot[:]),
	).Scan(&payouts).Error)
	r.Equal("150", payouts.Total)
}

// TestVoterFanOut checks the parallel arrays land as one row per voter with the
// pairing intact.
func TestVoterFanOut(t *testing.T) {
	resetTestDB(t)
	r := require.New(t)

	delegate := identityset.Address(1)
	voters := []address.Address{identityset.Address(3), identityset.Address(4)}
	args := chunkArgs(9, delegate, hash.Hash256b([]byte("s")), big.NewInt(1), voters,
		[]*big.Int{big.NewInt(11), big.NewInt(22)})
	// voter[1] redirected its reward elsewhere.
	args.Recipients = []address.Address{voters[0], identityset.Address(8)}
	// voter[0] compounded into bucket 0 -- a real bucket. voter[1] did not,
	// and also carries bucket id 0. Only Compounded distinguishes them.
	args.CompoundBucketIDs = []uint64{0, 0}
	args.Compounded = []bool{true, false}

	runBlock(t, blockWithLogs(t, 200, distributedLog(t, args)))

	var rows []models.IIP59VoterReward
	r.NoError(testGormDB.Order("voter").Find(&rows).Error)
	r.Len(rows, 2)

	byVoter := map[string]models.IIP59VoterReward{}
	for _, row := range rows {
		byVoter[row.Voter] = row
	}

	v0 := byVoter[voters[0].String()]
	r.Equal("11", v0.Amount.String())
	r.Equal(voters[0].String(), v0.Recipient, "no destination set: recipient is the voter")
	r.True(v0.Compounded)
	r.Equal(uint64(0), v0.CompoundBucketID, "bucket 0 is a real bucket")

	v1 := byVoter[voters[1].String()]
	r.Equal("22", v1.Amount.String())
	r.Equal(identityset.Address(8).String(), v1.Recipient,
		"destination set: recipient differs from voter, and reconciliation must follow recipient")
	r.False(v1.Compounded)
	r.Equal(uint64(0), v1.CompoundBucketID,
		"bucket id 0 on a non-compounded row must not be readable as compounded")
}

// TestIgnoresLogsFromOtherAddresses pins the address filter. Unpack cannot tell
// who emitted a log, so without this filter any contract emitting the same
// topic would be indexed as a protocol payout.
func TestIgnoresLogsFromOtherAddresses(t *testing.T) {
	resetTestDB(t)
	r := require.New(t)

	args := chunkArgs(1, identityset.Address(1), hash.Hash256b([]byte("x")), big.NewInt(1),
		[]address.Address{identityset.Address(3)}, []*big.Int{big.NewInt(7)})
	impostor := distributedLog(t, args)
	impostor.Address = identityset.Address(15).String()

	runBlock(t, blockWithLogs(t, 300, impostor))

	var count int64
	r.NoError(testGormDB.Model(&models.IIP59DelegateDistribution{}).Count(&count).Error)
	r.Zero(count, "a same-topic log from a non-rewarding address must be ignored")
}

// TestReplayIsIdempotent covers a block being processed twice, which happens on
// restart and reorg handling.
func TestReplayIsIdempotent(t *testing.T) {
	resetTestDB(t)
	r := require.New(t)

	blk := blockWithLogs(t, 400, distributedLog(t, chunkArgs(
		3, identityset.Address(1), hash.Hash256b([]byte("y")), big.NewInt(5),
		[]address.Address{identityset.Address(3), identityset.Address(4)},
		[]*big.Int{big.NewInt(1), big.NewInt(2)},
	)))

	runBlock(t, blk)
	runBlock(t, blk)

	var summaries, payouts int64
	r.NoError(testGormDB.Model(&models.IIP59DelegateDistribution{}).Count(&summaries).Error)
	r.NoError(testGormDB.Model(&models.IIP59VoterReward{}).Count(&payouts).Error)
	r.EqualValues(1, summaries)
	r.EqualValues(2, payouts)
}

// actionPayload mirrors iotex-core's unexported action.actionPayload. Declared
// here so this helper can be generic over the two settings actions; its method
// set is a superset of what EnvelopeBuilder.SetAction requires, which is what
// makes the assignment legal across the package boundary.
type actionPayload interface {
	IntrinsicGas() (uint64, error)
	SanityCheck() error
	FillAction(*iotextypes.ActionCore)
}

// signedAction wraps an action payload into a signed envelope so the plugin
// derives a sender from it the same way it does on-chain.
func signedAction(t *testing.T, payload actionPayload, key crypto.PrivateKey) *action.SealedEnvelope {
	t.Helper()
	elp := (&action.EnvelopeBuilder{}).
		SetNonce(1).
		SetGasLimit(100000).
		SetGasPrice(big.NewInt(1)).
		SetAction(payload).
		Build()
	selp, err := action.Sign(elp, key)
	require.NoError(t, err)
	return selp
}

func blockWithAction(t *testing.T, height uint64, selp *action.SealedEnvelope, status uint64) *block.Block {
	t.Helper()
	actHash, err := selp.Hash()
	require.NoError(t, err)
	receipt := &action.Receipt{
		Status:      status,
		BlockHeight: height,
		ActionHash:  actHash,
	}
	tb := block.NewTestingBuilder()
	blk, err := tb.SetPrevBlockHash(hash.ZeroHash256).
		SetHeight(height).
		SetTimeStamp(time.Unix(int64(height), 0)).
		AddActions(selp).
		SetReceipts([]*action.Receipt{receipt}).
		SignAndBuild(identityset.PrivateKey(0))
	require.NoError(t, err)
	return &blk
}

// TestSettingsActionsIndexed covers the two IIP-59 settings actions. They are
// history rather than current state -- a delegate can flip opt-in repeatedly --
// so the table keeps every change and the newest block_height wins on read.
func TestSettingsActionsIndexed(t *testing.T) {
	resetTestDB(t)
	r := require.New(t)

	signer := identityset.PrivateKey(3)
	sender := identityset.Address(3)
	candidate := identityset.Address(1)

	optIn := signedAction(t, action.NewSetVoterRewardOptIn(candidate.Bytes(), true), signer)
	runBlock(t, blockWithAction(t, 600, optIn, uint64(iotextypes.ReceiptStatus_Success)))

	optOut := signedAction(t, action.NewSetVoterRewardOptIn(candidate.Bytes(), false), signer)
	runBlock(t, blockWithAction(t, 601, optOut, uint64(iotextypes.ReceiptStatus_Success)))

	var rows []models.IIP59DelegateOptIn
	r.NoError(testGormDB.Order("block_height").Find(&rows).Error)
	r.Len(rows, 2, "both changes are kept; this table is a log, not current state")
	r.Equal(candidate.String(), rows[0].Candidate)
	r.Equal(sender.String(), rows[0].Sender)
	r.True(rows[0].OptIn)
	r.False(rows[1].OptIn)

	dest := signedAction(t, action.NewSetVoterRewardDestination(identityset.Address(9).Bytes()), signer)
	runBlock(t, blockWithAction(t, 602, dest, uint64(iotextypes.ReceiptStatus_Success)))

	var dests []models.IIP59VoterDestination
	r.NoError(testGormDB.Find(&dests).Error)
	r.Len(dests, 1)
	r.Equal(sender.String(), dests[0].Voter)
	r.Equal(identityset.Address(9).String(), dests[0].Recipient)
}

// TestFailedSettingsActionNotIndexed pins the receipt-status filter. A reverted
// opt-in never took effect, so recording it would make the "latest row wins"
// read return a setting the chain does not have.
func TestFailedSettingsActionNotIndexed(t *testing.T) {
	resetTestDB(t)
	r := require.New(t)

	failed := signedAction(t,
		action.NewSetVoterRewardOptIn(identityset.Address(1).Bytes(), true),
		identityset.PrivateKey(3))
	runBlock(t, blockWithAction(t, 700, failed, uint64(iotextypes.ReceiptStatus_Failure)))

	var count int64
	r.NoError(testGormDB.Model(&models.IIP59DelegateOptIn{}).Count(&count).Error)
	r.Zero(count, "a reverted action must not appear as history")
}

// TestBlockWithNoIIP59DataIsNoOp guards the common case: almost every block has
// nothing for this plugin.
func TestBlockWithNoIIP59DataIsNoOp(t *testing.T) {
	resetTestDB(t)
	r := require.New(t)

	runBlock(t, blockWithLogs(t, 500))

	var count int64
	r.NoError(testGormDB.Model(&models.IIP59DelegateDistribution{}).Count(&count).Error)
	r.Zero(count)
}
