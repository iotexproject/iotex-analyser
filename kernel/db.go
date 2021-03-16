package kernel

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/pkg/log"
)

var db *sql.DB
var dbOnce sync.Once

func GetDB() *sql.DB {
	dbOnce.Do(func() {
		var err error
		if db, err = sql.Open("mysql", config.Default.Database.Dsn); err != nil {
			log.L().Fatal(err.Error())
		}
		db.SetMaxOpenConns(20)
		db.SetMaxIdleConns(0)
		db.SetConnMaxLifetime(time.Nanosecond)
	})
	return db
}

func InsertTableData(tx *sql.Tx, table string, data map[string]interface{}) error {
	var cols, vals []string
	var x []interface{}
	for k, v := range data {
		cols, vals = append(cols, k), append(vals, "?")
		x = append(x, v)
	}
	sqlStr := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, table, strings.Join(cols, ","), strings.Join(vals, ","))
	stmt, err := tx.Prepare(sqlStr)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if _, err := stmt.Exec(x...); err != nil {
		return err
	}
	return nil

}

func Transaction(txFunc func(*sql.Tx) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	err = txFunc(tx)
	if err != nil {
		return
	}
	err = tx.Commit()
	return err
}

func UpdateIndexHeight(tx *sql.Tx, name string, height uint64) error {
	if _, err := tx.Exec("INSERT INTO index_heights (`name`, `height`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `height` = ?", name, height, height); err != nil {
		return err
	}
	return nil
}
