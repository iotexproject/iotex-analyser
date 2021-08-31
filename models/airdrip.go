package models

import "github.com/shopspring/decimal"

type (
	AirdripRegistration struct {
		ID          uint64 `gorm:"primary_key;" sql:"type:bigint"`
		BlockHeight uint64 `gorm:"unsigned;index" sql:"type:bigint"`
		User        string `gorm:"size:42;not null"`
		ExpireAt    uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	}

	AirdripExchange struct {
		ID          uint64          `gorm:"primary_key;" sql:"type:bigint"`
		BlockHeight uint64          `gorm:"unsigned;index" sql:"type:bigint"`
		User        string          `gorm:"size:42;index;not null"`
		Amount      decimal.Decimal `gorm:"type:decimal(60,0);not null"`
		Rate        uint64          `gorm:"unsigned"`
	}

	AirdripClaim struct {
		ID          uint64          `gorm:"primary_key" sql:"type:bigint"`
		BlockHeight uint64          `gorm:"unsigned;index" sql:"type:bigint"`
		User        string          `gorm:"size:42;not null"`
		Amount      decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	}

	AirdripRedemption struct {
		ID          uint64          `gorm:"primary_key" sql:"type:bigint"`
		BlockHeight uint64          `gorm:"unsigned;index" sql:"type:bigint"`
		Asset       string          `gorm:"size:42;index"`
		Amount      decimal.Decimal `gorm:"type:decimal(60,0);not null"`
		Points      decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	}

	AirdripAddAsset struct {
		ID          uint64          `gorm:"primary_key" sql:"type:bigint"`
		BlockHeight uint64          `gorm:"unsigned;index" sql:"type:bigint"`
		Provider    string          `gorm:"size:42;index"`
		Asset       string          `gorm:"size:42;index"`
		Amount      decimal.Decimal `gorm:"type:decimal(60,0);not null"`
		EndBlock    uint64          `gorm:"unsigned;index"`
		Konstante   uint64          `gorm:"unsigned"`
	}

	AirdripDrip struct {
		ID          uint64          `gorm:"primary_key" sql:"type:bigint"`
		BlockHeight uint64          `gorm:"unsigned;index" sql:"type:bigint"`
		Asset       string          `gorm:"size:42;index"`
		Volume      decimal.Decimal `gorm:"type:decimal(60,0);not null"`
	}

	AirdripTerm struct {
		ID          uint64 `gorm:"primary_key"`
		BlockHeight uint64 `gorm:"unsigned;index" sql:"type:bigint"`
		Number      uint64 `gorm:"unsigned;index" sql:"type:bigint"`
		Height      uint64 `gorm:"unsigned;index" sql:"type:bigint"`
	}
)

func (AirdripRegistration) TableName() string {
	return "airdrip_registrations"
}

func (AirdripExchange) TableName() string {
	return "airdrip_exchanges"
}

func (AirdripClaim) TableName() string {
	return "airdrip_claims"
}

func (AirdripRedemption) TableName() string {
	return "airdrip_redemptions"
}

func (AirdripAddAsset) TableName() string {
	return "airdrip_assets"
}

func (AirdripDrip) TableName() string {
	return "airdrip_drips"
}

func (AirdripTerm) TableName() string {
	return "airdrip_terms"
}
