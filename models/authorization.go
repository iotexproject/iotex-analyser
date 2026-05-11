package models

type Authorization struct {
	ID          uint64  `gorm:"primary_key;" sql:"type:bigint"`
	ActionHash  string  `gorm:"size:64;not null;index:,type:hash"`
	BlockHeight uint64  `gorm:"unsigned;index" sql:"type:bigint"`
	Index       int     `gorm:"type:int4;not null;default:0"`
	ChainID     string  `gorm:"size:66;not null;default:''"`
	Address     string  `gorm:"size:42;not null;default:'';index:,type:hash"` // delegate contract
	Nonce       string  `gorm:"size:66;not null;default:''"`
	YParity     string  `gorm:"size:66;not null;default:''"`
	R           string  `gorm:"size:66;not null;default:''"`
	S           string  `gorm:"size:66;not null;default:''"`
	Authority   string  `gorm:"size:42;not null;default:'';index:,type:hash"` // recovered signer
	Valid       *bool   `gorm:"default:null"` // TODO: set via eth_getCode(authority, blockHeight)
}

func (Authorization) TableName() string {
	return "authorization"
}
