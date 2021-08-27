package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Action struct {
	ID                 uint64          `gorm:"primary_key;" sql:"type:bigint"`
	ActionHash         string          `gorm:"size:64;not null;index:,length:9"`
	ActionType         string          `gorm:"size:32;not null;index"`
	BlockHeight        uint64          `gorm:"unsigned;index" sql:"type:bigint"`
	Sender             string          `gorm:"size:42;not null;default:'';index:,length:9"`
	Recipient          string          `gorm:"size:42;not null;default:'';index:,length:9"`
	GasPrice           decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	GasLimit           uint64          `gorm:"type:int4;unsigned;not null;default:0"`
	Nonce              uint64          `gorm:"type:int8;unsigned;not null;default:0"`
	Amount             decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	GasConsumed        uint64          `gorm:"type:int4;unsigned;not null;default:0"`
	ContractAddress    string          `gorm:"size:42;not null;default:'';"`
	Status             uint64          `gorm:"type:int2;unsigned;not null;default:0"`
	ExecutionRevertMsg string          `gorm:"size:255;not null;default:''"`
	Timestamp          time.Time       `gorm:"type:timestamp;index:,expression:(timestamp::date)"`
	//CREATE INDEX actions_timestamp_date_idx ON actions ((timestamp::DATE));
	//use index select timestamp::date, count(1) from actions where timestamp::date between '2021-08-15' and '2021-08-28' group by 1 order by 1;
}

func (Action) TableName() string {
	return "actions"
}
