package models

import "time"

// InscriptionRaw all actions that can convert 'input_data' into UTF-8
type InscriptionRaw struct {
	ID               uint64    `gorm:"primary_key" sql:"type:bigint"`
	BlockHeight      uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash       string    `gorm:"size:64;not null;index:,length:9"`
	TransactionIndex uint64    `gorm:"unsigned" sql:"type:bigint"`
	Sender           string    `gorm:"size:42;not null;default:'';index"`
	Recipient        string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Timestamp        time.Time `gorm:"type:timestamp;"`

	RawData string `gorm:"type:text;not null"`
}

func (InscriptionRaw) TableName() string {
	return "inscription_raws"
}

// Inscription inscription protocol
type Inscription struct {
	ID               uint64 `gorm:"primary_key" sql:"type:bigint"`
	BlockHeight      uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash       string `gorm:"size:64;not null;index:,length:9"`
	TransactionIndex uint64 `gorm:"unsigned" sql:"type:bigint"`
	MIMEType         string `gorm:"type:text;not null"`
	Parameters       string `gorm:"type:text;not null"`
	Extension        string `gorm:"type:text;not null"`
	Data             string `gorm:"type:text;not null"`
}

func (Inscription) TableName() string {
	return "inscriptions"
}

// InscriptionTransfer EOA transfer & Contract transfer
type InscriptionTransfer struct {
	ID               uint64    `gorm:"primary_key" sql:"type:bigint"`
	BlockHeight      uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash       string    `gorm:"size:64;not null;index:,length:9"`
	TransactionIndex uint64    `gorm:"unsigned" sql:"type:bigint"`
	Sender           string    `gorm:"size:42;not null;default:'';index"`
	Recipient        string    `gorm:"size:42;not null;default:'';index:,length:9"`
	Timestamp        time.Time `gorm:"type:timestamp;"`
	InscriptionHash  string    `gorm:"size:64;not null;index:,length:9"`
}

func (InscriptionTransfer) TableName() string {
	return "inscription_transfers"
}

// InscriptionHolder inscription holder
type InscriptionHolder struct {
	ID              uint64    `gorm:"primary_key" sql:"type:bigint"`
	Owner           string    `gorm:"size:42;not null;index:,default:'';"`
	InscriptionHash string    `gorm:"size:64;not null;index:,length:9"`
	Timestamp       time.Time `gorm:"type:timestamp;"`
}

func (InscriptionHolder) TableName() string {
	return "inscription_holders"
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
	Method           string    `gorm:"size:42;not null;default:'';"`
}

func (InscriptionTokenTransaction) TableName() string {
	return "inscription_token_transfers"
}

// InscriptionToken token deploy
type InscriptionToken struct {
	ID          uint64    `gorm:"primary_key" sql:"type:bigint"`
	BlockHeight uint64    `gorm:"unsigned;index" sql:"type:bigint"`
	ActionHash  string    `gorm:"size:64;index;not null;,length:9"`
	Owner       string    `gorm:"size:42;index;not null;default:'';"`
	Protocol    string    `gorm:"size:255;index;not null;default:'';"`
	Tick        string    `gorm:"size:255;index;not null;default:'';"`
	Op          string    `gorm:"size:255;not null;default:'';"`
	Max         uint64    `gorm:"unsigned" sql:"type:bigint"`
	Limit       uint64    `gorm:"unsigned" sql:"type:bigint"`
	Mint        uint64    `gorm:"unsigned" sql:"type:bigint"`
	Description string    `gorm:"size:255;not null;default:'';"`
	Verified    bool      `gorm:"type:bool;not null;default:true"`
	Timestamp   time.Time `gorm:"type:timestamp;"`
}

func (InscriptionToken) TableName() string {
	return "inscription_tokens"
}

type InscriptionTokenHolder struct {
	ID        uint64    `gorm:"primary_key" sql:"type:bigint"`
	Owner     string    `gorm:"size:42;index;not null;default:'';"`
	Protocol  string    `gorm:"size:255;index;not null;default:'';"`
	Tick      string    `gorm:"size:255;index;not null;default:'';"`
	Amount    uint64    `gorm:"unsigned" sql:"type:bigint"`
	Timestamp time.Time `gorm:"type:timestamp;"`
}

func (InscriptionTokenHolder) TableName() string {
	return "inscription_token_holders"
}
