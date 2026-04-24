package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	stdck "github.com/ClickHouse/clickhouse-go/v2"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/iotexproject/go-pkgs/crypto"
	"github.com/iotexproject/go-pkgs/hash"
	dbpkg "github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	testGormDB     *gorm.DB
	clickhouseDB   *gorm.DB
	clickhouseName string
	clickhousePort string
)

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().Port(9992),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=9992 sslmode=disable"
	var err error
	testGormDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		fmt.Printf("failed to connect embedded postgres: %v\n", err)
		_ = pg.Stop()
		os.Exit(1)
	}
	dbpkg.SetDB(testGormDB)
	if err := testGormDB.AutoMigrate(&dbpkg.IndexHeight{}, &models.Candidate{}); err != nil {
		fmt.Printf("failed to migrate postgres tables: %v\n", err)
		_ = pg.Stop()
		os.Exit(1)
	}

	clickhouseName = fmt.Sprintf("staking-actions-ch-test-%d", time.Now().UnixNano())
	clickhousePort, err = startClickHouse(clickhouseName)
	if err != nil {
		fmt.Printf("failed to start clickhouse container: %v\n", err)
		_ = pg.Stop()
		os.Exit(1)
	}
	clickhouseDB, err = connectClickHouse(clickhousePort)
	if err != nil {
		_ = stopClickHouse(clickhouseName)
		_ = pg.Stop()
		fmt.Printf("failed to connect clickhouse: %v\n", err)
		os.Exit(1)
	}
	chDB = clickhouseDB
	if err := recreateStakingActionsTable(); err != nil {
		_ = stopClickHouse(clickhouseName)
		_ = pg.Stop()
		fmt.Printf("failed to create clickhouse table: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	_ = stopClickHouse(clickhouseName)
	_ = pg.Stop()
	os.Exit(code)
}

func TestStakingActionChPutBlockLifecycle(t *testing.T) {
	require := require.New(t)
	resetDBs(t)

	addrA := identityset.Address(1).String()
	addrB := identityset.Address(2).String()
	addrC := identityset.Address(3).String()
	cand1Owner := identityset.Address(11).String()
	cand2Owner := identityset.Address(12).String()

	seedCandidate(t, "cand1", cand1Owner, 1)
	seedCandidate(t, "cand2", cand2Owner, 1)
	seedIndexHeight(t, "staking_actions_ch")

	blocks := []*block.Block{
		mustBuildBlock(t, 1,
			mustSignedCreateStake(t, 1, "cand1", "100", 30, true, identityset.PrivateKey(1)),
			receiptFor(t, mustSignedCreateStake(t, 1, "cand1", "100", 30, true, identityset.PrivateKey(1)), uint64(iotextypes.ReceiptStatus_Success), bucketLog(100)),
		),
		mustBuildBlock(t, 2,
			mustSignedDepositToStake(t, 2, 100, "20", identityset.PrivateKey(1)),
			receiptFor(t, mustSignedDepositToStake(t, 2, 100, "20", identityset.PrivateKey(1)), uint64(iotextypes.ReceiptStatus_Success)),
		),
		mustBuildBlock(t, 3,
			mustSignedTransferStake(t, 3, addrB, 100, identityset.PrivateKey(1)),
			receiptFor(t, mustSignedTransferStake(t, 3, addrB, 100, identityset.PrivateKey(1)), uint64(iotextypes.ReceiptStatus_Success)),
		),
		mustBuildBlock(t, 4,
			mustSignedRestake(t, 4, 100, 60, false, identityset.PrivateKey(2)),
			receiptFor(t, mustSignedRestake(t, 4, 100, 60, false, identityset.PrivateKey(2)), uint64(iotextypes.ReceiptStatus_Success)),
		),
		mustBuildBlock(t, 5,
			mustSignedChangeCandidate(t, 5, "cand2", 100, identityset.PrivateKey(2)),
			receiptFor(t, mustSignedChangeCandidate(t, 5, "cand2", 100, identityset.PrivateKey(2)), uint64(iotextypes.ReceiptStatus_Success)),
		),
		mustBuildBlock(t, 6,
			mustSignedDepositToStake(t, 6, 100, "30", identityset.PrivateKey(2)),
			receiptFor(t, mustSignedDepositToStake(t, 6, 100, "30", identityset.PrivateKey(2)), uint64(iotextypes.ReceiptStatus_Success)),
		),
		mustBuildBlock(t, 7,
			mustSignedReclaimStake(t, 7, 100, identityset.PrivateKey(2)),
			receiptFor(t, mustSignedReclaimStake(t, 7, 100, identityset.PrivateKey(2)), uint64(iotextypes.ReceiptStatus_Success)),
		),
		mustBuildBlock(t, 8,
			mustSignedCandidateRegister(t, 8, "cand3", addrC, "500", 91, true, identityset.PrivateKey(3)),
			receiptFor(t, mustSignedCandidateRegister(t, 8, "cand3", addrC, "500", 91, true, identityset.PrivateKey(3)), uint64(iotextypes.ReceiptStatus_Success), bucketLog(200)),
		),
		mustBuildBlock(t, 9,
			mustSignedDepositToStake(t, 9, 100, "999", identityset.PrivateKey(2)),
			receiptFor(t, mustSignedDepositToStake(t, 9, 100, "999", identityset.PrivateKey(2)), uint64(iotextypes.ReceiptStatus_Failure)),
		),
	}

	p := &stakingActionChPlugin{}
	for _, blk := range blocks {
		require.NoError(p.PutBlock(context.Background(), blk))
	}

	rows := loadRows(t)
	require.Len(rows, 10)
	assertRow(t, rows[0], 1, 0, 100, addrA, cand1Owner, "100", "StakeCreate", true, 30)
	assertRow(t, rows[1], 2, 0, 100, addrA, cand1Owner, "20", "DepositToStake", true, 30)
	assertRow(t, rows[2], 3, 0, 100, addrA, cand1Owner, "-120", "TransferStake", true, 30)
	assertRow(t, rows[3], 3, 0, 100, addrB, cand1Owner, "120", "TransferStake", true, 30)
	assertRow(t, rows[4], 4, 0, 100, addrB, cand1Owner, "0", "Restake", false, 60)
	// Current plugin behavior reconstructs bucket info from the latest positive-amount row,
	// so actions after Restake still inherit the pre-Restake auto-stake/duration values.
	assertRow(t, rows[5], 5, 0, 100, addrB, cand1Owner, "-120", "ChangeCandidate", true, 30)
	assertRow(t, rows[6], 5, 0, 100, addrB, cand2Owner, "120", "ChangeCandidate", true, 30)
	assertRow(t, rows[7], 6, 0, 100, addrB, cand2Owner, "30", "DepositToStake", true, 30)
	assertRow(t, rows[8], 7, 0, 100, addrB, cand2Owner, "-150", "Unstake", true, 30)
	assertRow(t, rows[9], 8, 0, 200, addrC, addrC, "500", "CandidateRegister", true, 91)

	height, err := dbpkg.GetIndexHeight("staking_actions_ch")
	require.NoError(err)
	require.Equal(uint64(9), height)
}

func TestStakingActionChPutBlockRestakeFixBeforeGreenland(t *testing.T) {
	require := require.New(t)
	resetDBs(t)

	addrA := identityset.Address(1).String()
	cand1Owner := identityset.Address(11).String()
	seedCandidate(t, "cand1", cand1Owner, 1)
	seedIndexHeight(t, "staking_actions_ch")

	create := mustSignedCreateStake(t, 1, "cand1", "100", 30, true, identityset.PrivateKey(1))
	unstake := mustSignedReclaimStake(t, 2, 300, identityset.PrivateKey(1))
	restake := mustSignedRestake(t, 3, 300, 91, true, identityset.PrivateKey(1))

	blocks := []*block.Block{
		mustBuildBlock(t, 1, create, receiptFor(t, create, uint64(iotextypes.ReceiptStatus_Success), bucketLog(300))),
		mustBuildBlock(t, 2, unstake, receiptFor(t, unstake, uint64(iotextypes.ReceiptStatus_Success))),
		mustBuildBlock(t, 3, restake, receiptFor(t, restake, uint64(iotextypes.ReceiptStatus_Success))),
	}

	p := &stakingActionChPlugin{}
	for _, blk := range blocks {
		require.NoError(p.PutBlock(context.Background(), blk))
	}

	rows := loadRows(t)
	require.Len(rows, 3)
	assertRow(t, rows[0], 1, 0, 300, addrA, cand1Owner, "100", "StakeCreate", true, 30)
	assertRow(t, rows[1], 2, 0, 300, addrA, cand1Owner, "-100", "Unstake", true, 30)
	assertRow(t, rows[2], 3, 0, 300, addrA, cand1Owner, "100", "Restake", true, 91)
}

func TestStakingActionChPutBlocksMatchesSequential(t *testing.T) {
	require := require.New(t)
	resetDBs(t)

	addrB := identityset.Address(2).String()
	cand1Owner := identityset.Address(11).String()
	cand2Owner := identityset.Address(12).String()
	seedCandidate(t, "cand1", cand1Owner, 1)
	seedCandidate(t, "cand2", cand2Owner, 1)
	seedIndexHeight(t, "staking_actions_ch")

	create := mustSignedCreateStake(t, 1, "cand1", "100", 30, true, identityset.PrivateKey(1))
	deposit := mustSignedDepositToStake(t, 2, 100, "20", identityset.PrivateKey(1))
	transfer := mustSignedTransferStake(t, 3, addrB, 100, identityset.PrivateKey(1))
	change := mustSignedChangeCandidate(t, 4, "cand2", 100, identityset.PrivateKey(2))

	blocks := []*block.Block{
		mustBuildBlock(t, 1, create, receiptFor(t, create, uint64(iotextypes.ReceiptStatus_Success), bucketLog(100))),
		mustBuildBlock(t, 2, deposit, receiptFor(t, deposit, uint64(iotextypes.ReceiptStatus_Success))),
		mustBuildBlock(t, 3, transfer, receiptFor(t, transfer, uint64(iotextypes.ReceiptStatus_Success))),
		mustBuildBlock(t, 4, change, receiptFor(t, change, uint64(iotextypes.ReceiptStatus_Success))),
	}

	seq := &stakingActionChPlugin{}
	for _, blk := range blocks {
		require.NoError(seq.PutBlock(context.Background(), blk))
	}
	seqRows := loadRows(t)
	seqHeight, err := dbpkg.GetIndexHeight("staking_actions_ch")
	require.NoError(err)

	resetClickHouseTable(t)
	resetIndexHeight(t, "staking_actions_ch")

	batch := &stakingActionChPlugin{}
	require.NoError(batch.PutBlocks(context.Background(), blocks))
	batchRows := loadRows(t)
	batchHeight, err := dbpkg.GetIndexHeight("staking_actions_ch")
	require.NoError(err)

	require.Equal(seqHeight, batchHeight)
	require.Equal(rowSnapshots(seqRows), rowSnapshots(batchRows))
}

func resetDBs(t *testing.T) {
	t.Helper()
	require.NoError(t, testGormDB.Exec("TRUNCATE candidate, index_heights RESTART IDENTITY CASCADE").Error)
	dbpkg.ClearIndexCache()
	resetClickHouseTable(t)
}

func resetClickHouseTable(t *testing.T) {
	t.Helper()
	require.NoError(t, chDB.Exec("TRUNCATE TABLE staking_actions").Error)
}

func seedIndexHeight(t *testing.T, name string) {
	t.Helper()
	require.NoError(t, testGormDB.Create(&dbpkg.IndexHeight{Name: name, Height: 0}).Error)
}

func resetIndexHeight(t *testing.T, name string) {
	t.Helper()
	require.NoError(t, testGormDB.Exec("DELETE FROM index_heights WHERE name = ?", name).Error)
	dbpkg.ClearIndexCache()
	seedIndexHeight(t, name)
}

func seedCandidate(t *testing.T, name, owner string, height uint64) {
	t.Helper()
	require.NoError(t, testGormDB.Create(&models.Candidate{
		BlockHeight:     height,
		Name:            name,
		OwnerAddress:    owner,
		OperatorAddress: owner,
		RewardAddress:   owner,
		CandidateID:     owner,
		Amount:          decimal.NewFromInt(0),
	}).Error)
}

func mustBuildBlock(t *testing.T, height uint64, selp *action.SealedEnvelope, receipt *action.Receipt) *block.Block {
	t.Helper()
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

func receiptFor(t *testing.T, selp *action.SealedEnvelope, status uint64, logs ...*action.Log) *action.Receipt {
	t.Helper()
	actHash, err := selp.Hash()
	require.NoError(t, err)
	r := &action.Receipt{
		Status:      status,
		ActionHash:  actHash,
		BlockHeight: 0,
	}
	return r.AddLogs(logs...)
}

func bucketLog(bucketID uint64) *action.Log {
	var topic hash.Hash256
	new(big.Int).SetUint64(bucketID).FillBytes(topic[:])
	return &action.Log{
		Address: StakingProtocolAddress,
		Topics:  []hash.Hash256{hash.ZeroHash256, topic},
	}
}

func mustSignedCreateStake(t *testing.T, nonce uint64, candidate, amount string, duration uint32, autoStake bool, key crypto.PrivateKey) *action.SealedEnvelope {
	t.Helper()
	selp, err := action.SignedCreateStake(nonce, candidate, amount, duration, autoStake, nil, 100000, big.NewInt(1), key)
	require.NoError(t, err)
	return selp
}

func mustSignedDepositToStake(t *testing.T, nonce, bucketID uint64, amount string, key crypto.PrivateKey) *action.SealedEnvelope {
	t.Helper()
	selp, err := action.SignedDepositToStake(nonce, bucketID, amount, nil, 100000, big.NewInt(1), key)
	require.NoError(t, err)
	return selp
}

func mustSignedTransferStake(t *testing.T, nonce uint64, voterAddress string, bucketID uint64, key crypto.PrivateKey) *action.SealedEnvelope {
	t.Helper()
	selp, err := action.SignedTransferStake(nonce, voterAddress, bucketID, nil, 100000, big.NewInt(1), key)
	require.NoError(t, err)
	return selp
}

func mustSignedRestake(t *testing.T, nonce, bucketID uint64, duration uint32, autoStake bool, key crypto.PrivateKey) *action.SealedEnvelope {
	t.Helper()
	selp, err := action.SignedRestake(nonce, bucketID, duration, autoStake, nil, 100000, big.NewInt(1), key)
	require.NoError(t, err)
	return selp
}

func mustSignedChangeCandidate(t *testing.T, nonce uint64, candidate string, bucketID uint64, key crypto.PrivateKey) *action.SealedEnvelope {
	t.Helper()
	selp, err := action.SignedChangeCandidate(nonce, candidate, bucketID, nil, 100000, big.NewInt(1), key)
	require.NoError(t, err)
	return selp
}

func mustSignedReclaimStake(t *testing.T, nonce, bucketID uint64, key crypto.PrivateKey) *action.SealedEnvelope {
	t.Helper()
	selp, err := action.SignedReclaimStake(false, nonce, bucketID, nil, 100000, big.NewInt(1), key)
	require.NoError(t, err)
	return selp
}

func mustSignedCandidateRegister(t *testing.T, nonce uint64, name, ownerAddress, amount string, duration uint32, autoStake bool, key crypto.PrivateKey) *action.SealedEnvelope {
	t.Helper()
	selp, err := action.SignedCandidateRegister(nonce, name, ownerAddress, ownerAddress, ownerAddress, amount, duration, autoStake, nil, 100000, big.NewInt(1), key)
	require.NoError(t, err)
	return selp
}

func startClickHouse(name string) (string, error) {
	run := exec.Command(
		"docker", "run", "-d", "--rm", "-P",
		"-e", "CLICKHOUSE_DB=default",
		"-e", "CLICKHOUSE_USER=default",
		"-e", "CLICKHOUSE_PASSWORD=testpass",
		"--name", name,
		"clickhouse/clickhouse-server:24.3",
	)
	if output, err := run.CombinedOutput(); err != nil {
		return "", fmt.Errorf("docker run: %w: %s", err, strings.TrimSpace(string(output)))
	}
	var port string
	for i := 0; i < 60; i++ {
		cmd := exec.Command("docker", "port", name, "9000/tcp")
		output, err := cmd.CombinedOutput()
		if err == nil {
			parts := strings.Split(strings.TrimSpace(string(output)), ":")
			if len(parts) == 2 && parts[1] != "" {
				port = parts[1]
				break
			}
		}
		time.Sleep(time.Second)
	}
	if port == "" {
		_ = stopClickHouse(name)
		return "", fmt.Errorf("failed to discover clickhouse port")
	}
	return port, nil
}

func stopClickHouse(name string) error {
	cmd := exec.Command("docker", "rm", "-f", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker rm: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func connectClickHouse(port string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("clickhouse://default:testpass@127.0.0.1:%s/default?dial_timeout=2s&read_timeout=30s&max_execution_time=60", port)
	var lastErr error
	for i := 0; i < 60; i++ {
		opt, err := stdck.ParseDSN(dsn)
		if err != nil {
			return nil, err
		}
		db := stdck.OpenDB(opt)
		gdb, err := gorm.Open(clickhouse.New(clickhouse.Config{
			Conn:                   db,
			DefaultTableEngineOpts: "ENGINE=MergeTree() ORDER BY (bucket_id, block_height, index)",
		}), &gorm.Config{})
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				return gdb, nil
			} else {
				lastErr = pingErr
			}
		} else {
			lastErr = err
		}
		time.Sleep(time.Second)
	}
	return nil, lastErr
}

func recreateStakingActionsTable() error {
	if err := chDB.Exec("DROP TABLE IF EXISTS staking_actions").Error; err != nil {
		return err
	}
	return chDB.Exec(`
		CREATE TABLE staking_actions
		(
			"block_height" UInt64,
			"index" Int64,
			"bucket_id" UInt64,
			"owner_address" String,
			"candidate" String,
			"amount" String,
			"act_type" String,
			"sender" String,
			"act_hash" String,
			"auto_stake" UInt8,
			"duration" UInt32,
			"timestamp" DateTime64(3)
		)
		ENGINE = MergeTree
		ORDER BY (bucket_id, block_height, "index")
	`).Error
}

func loadRows(t *testing.T) []StakingActions {
	t.Helper()
	var rows []StakingActions
	require.NoError(t, chDB.Order("block_height, index, bucket_id, act_type, amount, owner_address, candidate").Find(&rows).Error)
	return rows
}

func assertRow(t *testing.T, row StakingActions, height uint64, index int, bucketID uint64, owner, candidate, amount, actType string, autoStake bool, duration uint32) {
	t.Helper()
	require.Equal(t, height, row.BlockHeight)
	require.Equal(t, index, row.Index)
	require.Equal(t, bucketID, row.BucketID)
	require.Equal(t, owner, row.OwnerAddress)
	require.Equal(t, candidate, row.Candidate)
	require.Equal(t, amount, row.Amount.String())
	require.Equal(t, actType, row.ActType)
	require.Equal(t, autoStake, row.AutoStake)
	require.Equal(t, duration, row.Duration)
}

type rowSnapshot struct {
	BlockHeight  uint64
	Index        int
	BucketID     uint64
	OwnerAddress string
	Candidate    string
	Amount       string
	ActType      string
	Sender       string
	ActHash      string
	AutoStake    bool
	Duration     uint32
}

func rowSnapshots(rows []StakingActions) []rowSnapshot {
	snapshots := make([]rowSnapshot, 0, len(rows))
	for _, row := range rows {
		snapshots = append(snapshots, rowSnapshot{
			BlockHeight:  row.BlockHeight,
			Index:        row.Index,
			BucketID:     row.BucketID,
			OwnerAddress: row.OwnerAddress,
			Candidate:    row.Candidate,
			Amount:       row.Amount.String(),
			ActType:      row.ActType,
			Sender:       row.Sender,
			ActHash:      row.ActHash,
			AutoStake:    row.AutoStake,
			Duration:     row.Duration,
		})
	}
	return snapshots
}
