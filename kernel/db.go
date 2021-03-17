package kernel

import (
	"bytes"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

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
		cols, vals = append(cols, fmt.Sprintf("`%s`", k)), append(vals, "?")
		x = append(x, v)
	}
	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, table, strings.Join(cols, ","), strings.Join(vals, ","))
	stmt, err := tx.Prepare(query)
	if err != nil {
		return errors.Wrapf(err, "failed to prepare query: %s", queryDebug(query, x...))
	}
	defer stmt.Close()
	if _, err := stmt.Exec(x...); err != nil {
		return errors.Wrapf(err, "failed to exec query: %s", queryDebug(query, x...))
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

func queryDebug(query string, args ...interface{}) string {
	var buffer bytes.Buffer
	nArgs := len(args)
	for i, part := range strings.Split(query, "?") {
		buffer.WriteString(part)
		if i < nArgs {
			switch a := args[i].(type) {
			case uint64, int64:
				buffer.WriteString(fmt.Sprintf("%d", a))
			case bool:
				buffer.WriteString(fmt.Sprintf("%t", a))
			case sql.NullBool:
				if a.Valid {
					buffer.WriteString(fmt.Sprintf("%t", a.Bool))
				} else {
					buffer.WriteString("NULL")
				}
			case sql.NullInt64:
				if a.Valid {
					buffer.WriteString(fmt.Sprintf("%d", a.Int64))
				} else {
					buffer.WriteString("NULL")
				}
			case sql.NullString:
				if a.Valid {
					buffer.WriteString(fmt.Sprintf("%q", a.String))
				} else {
					buffer.WriteString("NULL")
				}
			case sql.NullFloat64:
				if a.Valid {
					buffer.WriteString(fmt.Sprintf("%f", a.Float64))
				} else {
					buffer.WriteString("NULL")
				}
			default:
				buffer.WriteString(fmt.Sprintf(`"%s"`, a))
			}
		}
	}
	return buffer.String()
}
