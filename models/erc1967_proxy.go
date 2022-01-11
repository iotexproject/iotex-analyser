package models

type Erc1967Proxy struct {
	ID            uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight   uint64 `gorm:"not null;unsigned;index" sql:"type:bigint"`
	ActionHash    string `gorm:"size:64;not null;index:,length:9"`
	ProxyAddress  string `gorm:"size:42;index;not null"`
	OriginAddress string `gorm:"size:42;index;not null"`
}

func (Erc1967Proxy) TableName() string {
	return "erc1967_proxy"
}
