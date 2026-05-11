package models

// Authorization stores one entry from an EIP-7702 SetCodeTx's auth list.
// (ActionHash, Index) is the natural composite unique key — Index is the
// position within the tx's auth list — and is enforced via uniqueIndex so
// idempotent backfills (clause.OnConflict{...DoNothing:true}) actually
// work.
type Authorization struct {
	ID          uint64 `gorm:"primary_key;" sql:"type:bigint"`
	ActionHash  string `gorm:"size:64;not null;uniqueIndex:idx_authorization_action_hash_index,priority:1"`
	BlockHeight uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	Index       int    `gorm:"type:int4;not null;default:0;uniqueIndex:idx_authorization_action_hash_index,priority:2"`
	ChainID     string `gorm:"size:66;not null;default:''"`
	Address     string `gorm:"size:42;not null;default:'';index:,type:hash"` // delegate contract
	Nonce       string `gorm:"size:66;not null;default:''"`
	YParity     string `gorm:"size:66;not null;default:''"`
	R           string `gorm:"size:66;not null;default:''"`
	S           string `gorm:"size:66;not null;default:''"`
	Authority   string `gorm:"size:42;not null;default:'';index:,type:hash"` // recovered signer
	// Valid is populated by kernel.ComputeAuthorizationValidity at index/backfill
	// time. nil means "not yet evaluated" (e.g. archive RPC was unavailable).
	Valid *bool `gorm:"default:null"`
}

func (Authorization) TableName() string {
	return "authorization"
}
