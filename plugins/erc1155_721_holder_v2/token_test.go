package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/iotexproject/go-pkgs/hash"
	iotexaddress "github.com/iotexproject/iotex-address/address"
	dbpkg "github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
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
		fmt.Printf("failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=9993 sslmode=disable"
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
	if err := testGormDB.AutoMigrate(&dbpkg.IndexHeight{}); err != nil {
		fmt.Printf("failed to migrate postgres tables: %v\n", err)
		_ = pg.Stop()
		os.Exit(1)
	}

	code := m.Run()
	_ = pg.Stop()
	os.Exit(code)
}

func TestContractTypeCacheShortCircuit(t *testing.T) {
	resetTestState(t)

	erc721Addr := ioAddressFromCommon(common.HexToAddress("0x1000000000000000000000000000000000000001"))
	erc1155Addr := ioAddressFromCommon(common.HexToAddress("0x2000000000000000000000000000000000000002"))
	unknownAddr := ioAddressFromCommon(common.HexToAddress("0x3000000000000000000000000000000000000003"))

	erc721Contract[erc721Addr] = struct{}{}
	nonErc1155Contract[erc721Addr] = struct{}{}
	nonErc721Contract[erc1155Addr] = struct{}{}
	erc1155Contract[erc1155Addr] = struct{}{}
	nonErc721Contract[unknownAddr] = struct{}{}
	nonErc1155Contract[unknownAddr] = struct{}{}

	ok, err := isErc721(erc721Addr)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = isErc1155(erc721Addr)
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = isErc721(erc1155Addr)
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = isErc1155(erc1155Addr)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = isErc721(unknownAddr)
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = isErc1155(unknownAddr)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestTokenPluginPutBlockERC721Lifecycle(t *testing.T) {
	p := newStartedPlugin(t)

	contract := ioAddressFromCommon(common.HexToAddress("0x4100000000000000000000000000000000000001"))
	markERC721Contract(contract)

	holderACommon := common.HexToAddress("0x5100000000000000000000000000000000000001")
	holderBCommon := common.HexToAddress("0x5200000000000000000000000000000000000002")
	holderA := ioAddressFromCommon(holderACommon)
	holderB := ioAddressFromCommon(holderBCommon)

	mintBlock := makeBlockWithReceipts(t, 1,
		receiptWithLogs(1, successStatus, erc721TransferLog(t, contract, common.Address{}, holderACommon, big.NewInt(11))),
		receiptWithLogs(2, successStatus, erc721TransferLog(t, contract, common.Address{}, holderBCommon, big.NewInt(99))),
		receiptWithLogs(3, 0, erc721TransferLog(t, contract, common.Address{}, holderBCommon, big.NewInt(12))),
	)
	require.NoError(t, p.PutBlock(context.Background(), mintBlock))
	assertHolders(t, []holderSnapshot{
		{ContractAddress: contract, Holder: holderA, ErcType: 721, TokenID: "11", TokenValue: "1"},
		{ContractAddress: contract, Holder: holderB, ErcType: 721, TokenID: "99", TokenValue: "1"},
	})

	transferBlock := makeBlockWithReceipts(t, 2,
		receiptWithLogs(4, successStatus, erc721TransferLog(t, contract, holderACommon, holderBCommon, big.NewInt(11))),
	)
	require.NoError(t, p.PutBlock(context.Background(), transferBlock))
	assertHolders(t, []holderSnapshot{
		{ContractAddress: contract, Holder: holderB, ErcType: 721, TokenID: "11", TokenValue: "1"},
		{ContractAddress: contract, Holder: holderB, ErcType: 721, TokenID: "99", TokenValue: "1"},
	})

	burnBlock := makeBlockWithReceipts(t, 3,
		receiptWithLogs(5, successStatus, erc721TransferLog(t, contract, holderBCommon, common.Address{}, big.NewInt(11))),
	)
	require.NoError(t, p.PutBlock(context.Background(), burnBlock))
	assertHolders(t, []holderSnapshot{
		{ContractAddress: contract, Holder: holderB, ErcType: 721, TokenID: "99", TokenValue: "1"},
	})

	height, err := dbpkg.GetIndexHeight(p.Name())
	require.NoError(t, err)
	require.Equal(t, uint64(3), height)
}

func TestTokenPluginPutBlockERC1155Lifecycle(t *testing.T) {
	p := newStartedPlugin(t)

	contract := ioAddressFromCommon(common.HexToAddress("0x6100000000000000000000000000000000000001"))
	markERC1155Contract(contract)

	holderACommon := common.HexToAddress("0x7100000000000000000000000000000000000001")
	holderBCommon := common.HexToAddress("0x7200000000000000000000000000000000000002")
	holderA := ioAddressFromCommon(holderACommon)
	holderB := ioAddressFromCommon(holderBCommon)

	require.NoError(t, p.PutBlock(context.Background(), makeBlockWithReceipts(t, 1,
		receiptWithLogs(10, successStatus, erc1155TransferSingleLog(t, contract, common.Address{}, common.Address{}, holderACommon, big.NewInt(1), big.NewInt(10))),
	)))
	assertHolders(t, []holderSnapshot{
		{ContractAddress: contract, Holder: holderA, ErcType: 1155, TokenID: "1", TokenValue: "10"},
	})

	require.NoError(t, p.PutBlock(context.Background(), makeBlockWithReceipts(t, 2,
		receiptWithLogs(11, successStatus, erc1155TransferSingleLog(t, contract, common.Address{}, common.Address{}, holderACommon, big.NewInt(1), big.NewInt(5))),
	)))
	assertHolders(t, []holderSnapshot{
		{ContractAddress: contract, Holder: holderA, ErcType: 1155, TokenID: "1", TokenValue: "15"},
	})

	require.NoError(t, p.PutBlock(context.Background(), makeBlockWithReceipts(t, 3,
		receiptWithLogs(12, successStatus, erc1155TransferSingleLog(t, contract, common.Address{}, holderACommon, holderBCommon, big.NewInt(1), big.NewInt(4))),
	)))
	assertHolders(t, []holderSnapshot{
		{ContractAddress: contract, Holder: holderA, ErcType: 1155, TokenID: "1", TokenValue: "11"},
		{ContractAddress: contract, Holder: holderB, ErcType: 1155, TokenID: "1", TokenValue: "4"},
	})

	require.NoError(t, p.PutBlock(context.Background(), makeBlockWithReceipts(t, 4,
		receiptWithLogs(13, successStatus, erc1155TransferSingleLog(t, contract, common.Address{}, holderBCommon, common.Address{}, big.NewInt(1), big.NewInt(4))),
	)))
	assertHolders(t, []holderSnapshot{
		{ContractAddress: contract, Holder: holderA, ErcType: 1155, TokenID: "1", TokenValue: "11"},
	})

	require.NoError(t, p.PutBlock(context.Background(), makeBlockWithReceipts(t, 5,
		receiptWithLogs(14, successStatus, erc1155TransferSingleLog(t, contract, common.Address{}, holderACommon, common.Address{}, big.NewInt(1), big.NewInt(11))),
	)))
	assertHolders(t, nil)

	height, err := dbpkg.GetIndexHeight(p.Name())
	require.NoError(t, err)
	require.Equal(t, uint64(5), height)
}

func TestTokenPluginPutBlockERC1155TransferBatch(t *testing.T) {
	p := newStartedPlugin(t)

	contract := ioAddressFromCommon(common.HexToAddress("0x8100000000000000000000000000000000000001"))
	markERC1155Contract(contract)

	holderACommon := common.HexToAddress("0x8200000000000000000000000000000000000001")
	holderCCommon := common.HexToAddress("0x8300000000000000000000000000000000000003")
	holderA := ioAddressFromCommon(holderACommon)
	holderC := ioAddressFromCommon(holderCCommon)

	require.NoError(t, p.PutBlock(context.Background(), makeBlockWithReceipts(t, 1,
		receiptWithLogs(20, successStatus, erc1155TransferBatchLog(t, contract, common.Address{}, common.Address{}, holderACommon, []*big.Int{big.NewInt(1), big.NewInt(2)}, []*big.Int{big.NewInt(10), big.NewInt(20)})),
	)))
	require.NoError(t, p.PutBlock(context.Background(), makeBlockWithReceipts(t, 2,
		receiptWithLogs(21, successStatus, erc1155TransferBatchLog(t, contract, common.Address{}, holderACommon, holderCCommon, []*big.Int{big.NewInt(1), big.NewInt(2)}, []*big.Int{big.NewInt(3), big.NewInt(20)})),
	)))

	assertHolders(t, []holderSnapshot{
		{ContractAddress: contract, Holder: holderA, ErcType: 1155, TokenID: "1", TokenValue: "7"},
		{ContractAddress: contract, Holder: holderC, ErcType: 1155, TokenID: "1", TokenValue: "3"},
		{ContractAddress: contract, Holder: holderC, ErcType: 1155, TokenID: "2", TokenValue: "20"},
	})
}

func TestTokenPluginPutBlockSameBlockStateChaining(t *testing.T) {
	p := newStartedPlugin(t)

	contract := ioAddressFromCommon(common.HexToAddress("0x9100000000000000000000000000000000000001"))
	markERC1155Contract(contract)

	holderACommon := common.HexToAddress("0x9200000000000000000000000000000000000001")
	holderBCommon := common.HexToAddress("0x9300000000000000000000000000000000000002")
	holderCCommon := common.HexToAddress("0x9400000000000000000000000000000000000003")
	holderB := ioAddressFromCommon(holderBCommon)
	holderC := ioAddressFromCommon(holderCCommon)

	blk := makeBlockWithReceipts(t, 1,
		receiptWithLogs(30, successStatus, erc1155TransferSingleLog(t, contract, common.Address{}, common.Address{}, holderACommon, big.NewInt(7), big.NewInt(10))),
		receiptWithLogs(31, successStatus, erc1155TransferSingleLog(t, contract, common.Address{}, holderACommon, holderBCommon, big.NewInt(7), big.NewInt(6))),
		receiptWithLogs(32, successStatus, erc1155TransferBatchLog(t, contract, common.Address{}, holderACommon, holderCCommon, []*big.Int{big.NewInt(7)}, []*big.Int{big.NewInt(4)})),
	)
	require.NoError(t, p.PutBlock(context.Background(), blk))

	assertHolders(t, []holderSnapshot{
		{ContractAddress: contract, Holder: holderB, ErcType: 1155, TokenID: "7", TokenValue: "6"},
		{ContractAddress: contract, Holder: holderC, ErcType: 1155, TokenID: "7", TokenValue: "4"},
	})
}

func newStartedPlugin(t *testing.T) *tokenPlugin {
	t.Helper()
	resetTestState(t)
	p := &tokenPlugin{}
	require.NoError(t, p.Start(context.Background()))
	return p
}

func resetTestState(t *testing.T) {
	t.Helper()
	require.NotNil(t, testGormDB)
	erc721Contract = make(map[string]struct{})
	nonErc721Contract = make(map[string]struct{})
	erc1155Contract = make(map[string]struct{})
	nonErc1155Contract = make(map[string]struct{})
	dbpkg.ClearIndexCache()
	require.NoError(t, testGormDB.Exec("DROP TABLE IF EXISTS "+Erc1155721Holder{}.TableName()).Error)
	require.NoError(t, testGormDB.Exec("DELETE FROM index_heights WHERE name = ?", (&tokenPlugin{}).Name()).Error)
}

func markERC721Contract(addr string) {
	erc721Contract[addr] = struct{}{}
	nonErc1155Contract[addr] = struct{}{}
}

func markERC1155Contract(addr string) {
	nonErc721Contract[addr] = struct{}{}
	erc1155Contract[addr] = struct{}{}
}

func makeBlockWithReceipts(t *testing.T, height uint64, receipts ...*action.Receipt) *block.Block {
	t.Helper()
	tb := block.TestingBuilder{}
	blk, err := tb.SetPrevBlockHash(hash.ZeroHash256).
		SetVersion(1).
		SetTimeStamp(time.Unix(int64(height), 0)).
		SetHeight(height).
		SignAndBuild(identityset.PrivateKey(0))
	require.NoError(t, err)
	blk.Receipts = receipts
	return &blk
}

func receiptWithLogs(seed byte, status uint64, logs ...*action.Log) *action.Receipt {
	var actionHash hash.Hash256
	actionHash[0] = seed
	actionHash[31] = seed
	r := &action.Receipt{
		Status:     status,
		ActionHash: actionHash,
	}
	return r.AddLogs(logs...)
}

func erc721TransferLog(t *testing.T, contract string, from, to common.Address, tokenID *big.Int) *action.Log {
	t.Helper()
	return &action.Log{
		Address: contract,
		Topics: []hash.Hash256{
			Transfer,
			topicFromCommonAddress(from),
			topicFromCommonAddress(to),
			topicFromBig(tokenID),
		},
	}
}

func erc1155TransferSingleLog(t *testing.T, contract string, operator, from, to common.Address, tokenID, value *big.Int) *action.Log {
	t.Helper()
	data := packEventData(t, erc1155ABI.Events["TransferSingle"].Inputs.NonIndexed(), tokenID, value)
	return &action.Log{
		Address: contract,
		Topics: []hash.Hash256{
			HashTransferSingle,
			topicFromCommonAddress(operator),
			topicFromCommonAddress(from),
			topicFromCommonAddress(to),
		},
		Data: data,
	}
}

func erc1155TransferBatchLog(t *testing.T, contract string, operator, from, to common.Address, ids, values []*big.Int) *action.Log {
	t.Helper()
	data := packEventData(t, erc1155ABI.Events["TransferBatch"].Inputs.NonIndexed(), ids, values)
	return &action.Log{
		Address: contract,
		Topics: []hash.Hash256{
			HashTransferBatch,
			topicFromCommonAddress(operator),
			topicFromCommonAddress(from),
			topicFromCommonAddress(to),
		},
		Data: data,
	}
}

func packEventData(t *testing.T, args abi.Arguments, values ...interface{}) []byte {
	t.Helper()
	data, err := args.Pack(values...)
	require.NoError(t, err)
	return data
}

func topicFromCommonAddress(addr common.Address) hash.Hash256 {
	var topic hash.Hash256
	copy(topic[:], common.BytesToHash(addr.Bytes()).Bytes())
	return topic
}

func topicFromBig(value *big.Int) hash.Hash256 {
	var topic hash.Hash256
	value.FillBytes(topic[:])
	return topic
}

func ioAddressFromCommon(addr common.Address) string {
	ioAddr, err := iotexaddress.FromBytes(addr.Bytes())
	if err != nil {
		panic(err)
	}
	return ioAddr.String()
}

type holderSnapshot struct {
	ContractAddress string
	Holder          string
	ErcType         uint16
	TokenID         string
	TokenValue      string
}

func loadHolders(t *testing.T) []holderSnapshot {
	t.Helper()
	var rows []Erc1155721Holder
	require.NoError(t, testGormDB.Order("contract_address, holder, token_id").Find(&rows).Error)
	snapshots := make([]holderSnapshot, 0, len(rows))
	for _, row := range rows {
		snapshots = append(snapshots, holderSnapshot{
			ContractAddress: row.ContractAddress,
			Holder:          row.Holder,
			ErcType:         row.ErcType,
			TokenID:         row.TokenID.String(),
			TokenValue:      row.TokenValue.String(),
		})
	}
	return snapshots
}

func assertHolders(t *testing.T, expected []holderSnapshot) {
	t.Helper()
	require.ElementsMatch(t, expected, loadHolders(t))
}
