package verifymigration

import (
	"fmt"
	"strconv"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

var VerifyMigration = &cli.Command{
	Name:  "verify_migration",
	Usage: "verify_migration <subcommand>",
	Subcommands: []*cli.Command{
		VerifyActionExecutionCmd,
		VerifyCandidateCmd,
	},
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
		&cli.StringFlag{
			Name:  "chdsn",
			Usage: "clickhouse DSN",
			Value: "clickhouse://username:password@127.0.0.1:9000/testnet",
		},
		&cli.StringFlag{
			Name:  "pgdsn",
			Usage: "postgres DSN",
			Value: "postgres://postgres:mysecretpassword@127.0.0.1:5432/testnet?sslmode=disable",
		},
	},
}

func connectDatabase(c *cli.Context) (driver.Conn, *gorm.DB, error) {
	chDSN := c.String("chdsn")
	pgDSN := c.String("pgdsn")
	// verify action_execution
	chConn, err := db.ConnectClickhouse(chDSN)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to connect clickhouse")
	}
	fmt.Println("clickhouse connected successfully", chDSN)
	pgCfg, err := pgconn.ParseConfig(pgDSN)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse postgres dsn")
	}
	config.Default.Database.Driver = "postgres"
	config.Default.Database.User = pgCfg.User
	config.Default.Database.Password = pgCfg.Password
	config.Default.Database.Host = pgCfg.Host
	config.Default.Database.Name = pgCfg.Database
	config.Default.Database.Port = strconv.FormatInt(int64(pgCfg.Port), 10)
	pg, err := db.Connect()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to connect postgres")
	}
	fmt.Println("postgres connected successfully", config.Default.Database.Host)
	return chConn, pg, nil
}
