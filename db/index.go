package db

import (
	"errors"
	"sync"

	"gorm.io/gorm"
)

type BaseModel struct {
	IndexName   string `gorm:"-"`
	IndexHeight uint64 `gorm:"-"`
}

type IndexHeight struct {
	ID     uint32 `gorm:"primary_key;auto_increment"`
	Name   string `gorm:"size:128;not null;unique"`
	Height uint64 `sql:"type:bigint"`
}

var indexCache sync.Map

// ClearIndexCache clears the in-memory index height cache, intended for testing only.
func ClearIndexCache() {
	indexCache.Range(func(k, _ any) bool {
		indexCache.Delete(k)
		return true
	})
}

func UpdateIndexHeightByTx(tx *gorm.DB, name string, height uint64) error {
	indexCache.Store(name, height)
	return tx.Model(&IndexHeight{}).Where("name = ?", name).UpdateColumn("height", height).Error
}

func UpdateIndexHeight(name string, height uint64) error {
	indexCache.Store(name, height)
	return db.Model(&IndexHeight{}).Where("name = ?", name).UpdateColumn("height", height).Error
}

func GetIndexHeight(name string) (uint64, error) {
	height, ok := indexCache.Load(name)
	if ok {
		return height.(uint64), nil
	}
	var m *IndexHeight
	idx, err := m.ByName(name)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return 0, err
		}
		m = &IndexHeight{
			Name: name,
		}
		return 0, db.Create(m).Error
	}
	indexCache.Store(name, idx.Height)
	return idx.Height, nil
}

// GetCommittedIndexHeight reads name's index height from the database, skipping the
// cache. UpdateIndexHeightByTx caches a height before its transaction commits, and keeps
// it if the transaction rolls back. A plugin that must not read past name's committed data
// reads name's height here. Unlike GetIndexHeight it does not create a missing row, and
// reports 0 for it.
func GetCommittedIndexHeight(name string) (uint64, error) {
	var idx IndexHeight
	err := db.Model(&IndexHeight{}).Where("name = ?", name).Take(&idx).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return idx.Height, err
}

func (m *IndexHeight) ByName(name string) (*IndexHeight, error) {
	var err error
	err = db.Model(m).Where("name = ?", name).Take(&m).Error
	if err != nil {
		return nil, err
	}
	return m, err
}
