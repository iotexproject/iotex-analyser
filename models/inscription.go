package models

import "time"

type InscriptionRaw struct {
	ID          uint64    `gorm:"primary_key" sql:"type:bigint"`
	BlockHeight uint64    `gorm:"unsigned" sql:"type:bigint"`
	ActionHash  string    `gorm:"size:64;not null;index:,length:9"`
	Sender      string    `gorm:"size:42;not null;default:'';"`
	Recipient   string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Timestamp   time.Time `gorm:"type:timestamp;"`

	RawData string `gorm:"type:text;not null;default:'';"`
}
