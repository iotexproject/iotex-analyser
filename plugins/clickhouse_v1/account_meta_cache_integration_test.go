package main

import (
	"fmt"
	"math/big"
	"net"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/iotexproject/iotex-analyser/config"
	dbpkg "github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var testGormDB *gorm.DB

func TestMain(m *testing.M) {
	port, err := getFreePort()
	if err != nil {
		panic(err)
	}
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().Port(uint32(port)),
	)
	if err := pg.Start(); err != nil {
		panic(err)
	}

	dsn := fmt.Sprintf("host=localhost user=postgres password=postgres dbname=postgres port=%d sslmode=disable", port)
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		_ = pg.Stop()
		panic(err)
	}
	testGormDB = gdb
	dbpkg.SetDB(testGormDB)
	kernel.Init(&config.Default)
	if err := testGormDB.AutoMigrate(&models.AccountMeta{}); err != nil {
		_ = pg.Stop()
		panic(err)
	}

	code := m.Run()
	_ = pg.Stop()
	os.Exit(code)
}

func getFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func TestLoadAccountContractFlagsWithEmbeddedPostgres(t *testing.T) {
	resetAccountMetaTable(t)
	rows := []models.AccountMeta{
		{Address: identityset.Address(1).String(), IsContract: true},
		{Address: identityset.Address(2).String(), IsContract: false},
	}
	require.NoError(t, testGormDB.Create(&rows).Error)

	flags, err := loadAccountContractFlags([]string{
		rows[0].Address,
		rows[1].Address,
		identityset.Address(3).String(),
	})
	require.NoError(t, err)
	require.True(t, flags[rows[0].Address])
	require.False(t, flags[rows[1].Address])
	require.False(t, flags[identityset.Address(3).String()])
}

func TestPutBlockLoadsContractFlagsFromEmbeddedPostgres(t *testing.T) {
	resetAccountMetaTable(t)
	contractAddr := identityset.Address(1).String()
	userAddr := identityset.Address(2).String()
	recipient := identityset.Address(3).String()
	require.NoError(t, testGormDB.Create(&models.AccountMeta{Address: contractAddr, IsContract: true}).Error)
	require.NoError(t, testGormDB.Create(&models.AccountMeta{Address: userAddr, IsContract: false}).Error)

	p := &clickhouseV1Plugin{
		accountContract: make(map[string]bool),
	}
	require.NoError(t, p.putBlock(nil, makeTxLogBlock(1,
		[]*action.TransactionLog{
			{Sender: contractAddr, Recipient: recipient, Amount: big.NewInt(10)},
			{Sender: userAddr, Recipient: recipient, Amount: big.NewInt(5)},
		},
	)))

	require.Len(t, p.transactionLogs, 2)
	require.True(t, p.transactionLogs[0].Internal)
	require.False(t, p.transactionLogs[1].Internal)
	require.True(t, p.accountContract[contractAddr])
	require.False(t, p.accountContract[userAddr])
}

func resetAccountMetaTable(t *testing.T) {
	t.Helper()
	require.NotNil(t, testGormDB)
	require.NoError(t, testGormDB.Exec("TRUNCATE TABLE account_meta RESTART IDENTITY").Error)
}
