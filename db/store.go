package db

import (
	"time"
)

type Store struct {
	ID       uint32 `gorm:"primary_key;auto_increment"`
	Key      string `gorm:"size:128;not null;unique"`
	Value    string `sql:"type:json"`
	CreateAt time.Time
	UpdateAt time.Time
}

func (Store) TableName() string {
	return "store"
}

func (s *Store) Save() error {
	var count int64
	err := db.Model(s).Where("key = ?", s.Key).Count(&count).Error
	if err != nil {
		return err
	}

	if count > 0 {
		return db.Model(s).Where("key = ?", s.Key).UpdateColumn("update_at", time.Now()).UpdateColumn("value", s.Value).Error
	}
	s.CreateAt = time.Now()
	return db.Create(s).Error
}
