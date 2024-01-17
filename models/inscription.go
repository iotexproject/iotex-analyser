package models

import "time"

// InscriptionRaw all actions that can convert 'input_data' into UTF-8
type InscriptionRaw struct {
	ID               uint64    `gorm:"primary_key" sql:"type:bigint"`
	BlockHeight      uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash       string    `gorm:"size:64;not null;index:,length:9"`
	TransactionIndex uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	Sender           string    `gorm:"size:42;not null;default:'';"`
	Recipient        string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Timestamp        time.Time `gorm:"type:timestamp;"`

	RawData string `gorm:"type:text;not null;default:'';"`
}

// Inscription inscription protocol
type Inscription struct {
	ID               uint64 `gorm:"primary_key" sql:"type:bigint"`
	BlockHeight      uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash       string `gorm:"size:64;not null;index:,length:9"`
	TransactionIndex uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	MIMEType         string `gorm:"type:text;not null;default:'';"`
	Parameters       string `gorm:"type:text;not null;default:'';"`
	Extension        string `gorm:"type:text;not null;default:'';"`
	Data             string `gorm:"type:text;not null;default:'';"`
}

// InscriptionTransfer EOA transfer & Contract transfer
type InscriptionTransfer struct {
	ID               uint64    `gorm:"primary_key" sql:"type:bigint"`
	BlockHeight      uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash       string    `gorm:"size:64;not null;index:,length:9"`
	TransactionIndex uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	Sender           string    `gorm:"size:42;not null;default:'';"`
	Recipient        string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Timestamp        time.Time `gorm:"type:timestamp;"`
	InscriptionHash  string    `gorm:"size:64;not null;index:,length:9"`
}

// InscriptionHolder inscription holder
type InscriptionHolder struct {
	ID              uint64    `gorm:"primary_key" sql:"type:bigint"`
	Owner           string    `gorm:"size:42;not null;index:,default:'';"`
	InscriptionHash string    `gorm:"size:64;not null;index:,length:9"`
	IsTransfer      bool      `gorm:"type:bool;not null;default:false"`
	Timestamp       time.Time `gorm:"type:timestamp;"`
}

// InscriptionTokenTransaction filter io-20 protocol from Inscription
type InscriptionTokenTransaction struct {
	ID               uint64    `gorm:"primary_key" sql:"type:bigint"`
	BlockHeight      uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash       string    `gorm:"size:64;not null;index:,length:9"`
	TransactionIndex uint64    `gorm:"unsigned" sql:"type:bigint"`
	Sender           string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Recipient        string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Timestamp        time.Time `gorm:"type:timestamp;"`
	//InscriptionHash  string    `gorm:"size:64;not null;index:,length:9"`
	Method string `gorm:"size:42;not null;default:'';"`
}

// InscriptionToken token deploy
type InscriptionToken struct {
	ID          uint64    `gorm:"primary_key" sql:"type:bigint"`
	BlockHeight uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash  string    `gorm:"size:64;index;not null;index:,length:9"`
	Owner       string    `gorm:"size:42;index;not null;default:'';"`
	P           string    `gorm:"size:42;index;not null;default:'';"`
	Tick        string    `gorm:"size:42;index;not null;default:'';"`
	Op          string    `gorm:"size:42;not null;default:'';"`
	Max         uint64    `gorm:"unsigned" sql:"type:bigint"`
	Lim         uint64    `gorm:"unsigned" sql:"type:bigint"`
	Mint        uint64    `gorm:"unsigned" sql:"type:bigint"`
	Description string    `gorm:"size:255;not null;default:'';"`
	Verified    bool      `gorm:"type:bool;not null;default:true"`
	Timestamp   time.Time `gorm:"type:timestamp;"`
}

type InscriptionTokenHolder struct {
	ID        uint64    `gorm:"primary_key" sql:"type:bigint"`
	Owner     string    `gorm:"size:42;index;not null;default:'';"`
	P         string    `gorm:"size:42;index;not null;default:'';"`
	Tick      string    `gorm:"size:42;index;not null;default:'';"`
	Amt       uint64    `gorm:"unsigned" sql:"type:bigint"`
	Timestamp time.Time `gorm:"type:timestamp;"`
}
