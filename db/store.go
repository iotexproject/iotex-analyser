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
	err := db.Model(s).Where(&Store{Key: s.Key}).Count(&count).Error
	if err != nil {
		return err
	}

	now := time.Now()
	if count > 0 {
		return db.Model(s).Where(&Store{Key: s.Key}).UpdateColumn("update_at", now).UpdateColumn("value", s.Value).Error
	}
	s.CreateAt = now
	s.UpdateAt = now
	return db.Create(s).Error
}
