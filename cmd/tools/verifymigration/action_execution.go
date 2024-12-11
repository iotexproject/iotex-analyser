package verifymigration

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

var VerifyActionExecutionCmd = &cli.Command{
	Name:        "action_execution",
	Usage:       "action_execution",
	Description: "action_execution will verify the clickhouse from postgres by blkNum",
	Action:      verifyActionExecution,
}

func verifyActionExecution(c *cli.Context) error {
	start := c.Uint64("start")
	end := c.Uint64("end")
	chDSN := c.String("chdsn")
	pgDSN := c.String("pgdsn")
	// verify action_execution
	chConn, err := db.ConnectClickhouse(chDSN)
	if err != nil {
		return errors.Wrap(err, "failed to connect clickhouse")
	}
	fmt.Println("clickhouse connected successfully", chDSN)
	pgCfg, err := pgconn.ParseConfig(pgDSN)
	if err != nil {
		return errors.Wrap(err, "failed to parse postgres dsn")
	}
	config.Default.Database.Driver = "postgres"
	config.Default.Database.User = pgCfg.User
	config.Default.Database.Password = pgCfg.Password
	config.Default.Database.Host = pgCfg.Host
	config.Default.Database.Name = pgCfg.Database
	config.Default.Database.Port = strconv.FormatInt(int64(pgCfg.Port), 10)
	pg, err := db.Connect()
	if err != nil {
		return errors.Wrap(err, "failed to connect postgres")
	}
	fmt.Println("postgres connected successfully", config.Default.Database.Host)

	batchSize := 2000
	for i := start; i <= end; {
		size := uint64(batchSize)
		if i+size-1 > end {
			size = end - i + 1
		}
		if err := verifyActionExecutionInRange(c.Context, pg, chConn, i, size); err != nil {
			return errors.Wrap(err, "failed to verify action_execution")
		}
		i = i + size
	}
	return nil
}

type actionExecution struct {
	ID                     uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight            uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash             string `gorm:"size:64;not null;index:,length:9"`
	Contract               string `gorm:"size:42;not null;default:'';index:,length:9"`
	ReceiptContractAddress string `gorm:"size:42;not null;default:'';index:,length:9"`
	Data                   []byte `gorm:"not null;"`
}

func (actionExecution) TableName() string {
	return "action_execution"
}

func verifyActionExecutionInRange(ctx context.Context, pg *gorm.DB, ch driver.Conn, start, size uint64) error {
	fmt.Println("\nverify action_execution from", start, "to", start+size-1, ":")
	ches := make([]models.ActionExecution, 0)
	err := ch.Select(ctx, &ches,
		"SELECT * FROM action_execution WHERE block_height >= ? AND block_height < ?",
		start, start+size)
	if err != nil {
		return errors.Wrap(err, "failed to select action_execution from clickhouse")
	}

	pges := make([]*actionExecution, 0)
	err = pg.Model(&actionExecution{}).Where("block_height >= ? AND block_height < ?", start, start+size).Order("id").Find(&pges).Error
	if err != nil {
		return errors.Wrap(err, "failed to select action_execution from postgres")
	}

	fmt.Printf("\tclickhouse count: %d, postgres count: %d\n", len(ches), len(pges))
	isConsistent := len(ches) == len(pges)
	chesMap := make(map[string]*models.ActionExecution)
	for _, ae := range ches {
		chesMap[ae.ActionHash] = &ae
	}
	for _, pge := range pges {
		che, ok := chesMap[pge.ActionHash]
		if !ok {
			fmt.Printf("\taction_execution not found in clickhouse: %s\n", pge.ActionHash)
			isConsistent = false
			continue
		}
		if pge.BlockHeight != che.BlockHeight ||
			pge.ActionHash != che.ActionHash ||
			!addressEqual(pge.Contract, che.Contract) ||
			!addressEqual(pge.ReceiptContractAddress, che.ReceiptContractAddress) ||
			string(pge.Data) != string(che.Data) {
			if pge.BlockHeight != che.BlockHeight {
				fmt.Printf("\tblock_height not match: %v, %v\n", pge.BlockHeight, che.BlockHeight)
			}
			if pge.ActionHash != che.ActionHash {
				fmt.Printf("\taction_hash not match: %v, %v\n", pge.ActionHash, che.ActionHash)
			}
			if !addressEqual(pge.Contract, che.Contract) {
				fmt.Printf("\tcontract not match: %v, %v\n", pge.Contract, che.Contract)
			}
			if !addressEqual(pge.ReceiptContractAddress, che.ReceiptContractAddress) {
				fmt.Printf("\treceipt_contract_address not match: %#v, %#v\n", pge.ReceiptContractAddress, che.ReceiptContractAddress)
			}
			if string(pge.Data) != string(che.Data) {
				fmt.Printf("\tdata not match: %v, %v\n", pge.Data, che.Data)
			}
			fmt.Printf("\taction_execution not match: %v, %v\n", pge, che)
			isConsistent = false
		}
	}
	if isConsistent {
		fmt.Println("\taction_execution is consistent")
	}
	return nil
}

var zeroAddress = string(make([]byte, 41))

func addressEqual(a, b string) bool {
	if a == zeroAddress {
		a = ""
	}
	if b == zeroAddress {
		b = ""
	}
	return a == b
}
