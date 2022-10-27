package models

import (
	"database/sql/driver"
	"encoding/json"
)

type Endorser []string

func (c Endorser) Value() (driver.Value, error) {
	b, err := json.Marshal(c)
	return string(b), err
}

func (c *Endorser) Scan(src any) error {
	return json.Unmarshal(src.([]byte), c)
}

type BlockFooter struct {
	BlockHeight uint64   `gorm:"primary_key;" sql:"type:bigint;"`
	Endorser    Endorser `gorm:"type:json"`
}

func (BlockFooter) TableName() string {
	return "block_footer"
}
