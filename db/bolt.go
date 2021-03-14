package db

import (
	"context"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

// error definition
var (
	ErrBucketNotExist = errors.New("bucket not exist in DB")
	ErrNotExist       = errors.New("not exist in DB")
	ErrIO             = errors.New("DB I/O operation error")
)

type (
	// KVStore is the interface of KV store
	KVStore interface {
		// Get gets a record by (namespace, key)
		Get(string, []byte) ([]byte, error)

		// Put insert or update a record identified by (namespace, key)
		Put(string, []byte, []byte) error

		// Delete deletes a record by (namespace, key)
		Delete(string, []byte) error

		// Start starts the db
		Start(context.Context) error

		// Stop stops the db
		Stop(context.Context) error
	}

	boltDB struct {
		db   *bolt.DB
		path string
	}
)

// NewBoltDB instantiates an boltDB with implements KVStore
func NewBoltDB(path string) KVStore {
	return &boltDB{
		db:   nil,
		path: path,
	}
}

// Start opens the boltDB (creates new file if not existing yet)
func (b *boltDB) Start(_ context.Context) error {
	db, err := bolt.Open(b.path, 0600, nil)
	if err != nil {
		return errors.Wrap(ErrIO, err.Error())
	}
	b.db = db
	return nil
}

// Stop closes the boltDB
func (b *boltDB) Stop(_ context.Context) error {
	if b.db != nil {
		if err := b.db.Close(); err != nil {
			return errors.Wrap(ErrIO, err.Error())
		}
	}
	return nil
}

// Get retrieves a record
func (b *boltDB) Get(namespace string, key []byte) ([]byte, error) {
	var value []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(namespace))
		if bucket == nil {
			return errors.Wrapf(ErrBucketNotExist, "bucket = %x doesn't exist", []byte(namespace))
		}
		v := bucket.Get(key)
		if v == nil {
			return errors.Wrapf(ErrNotExist, "key = %x doesn't exist", key)
		}
		value = make([]byte, len(v))
		// TODO: this is not an efficient way of passing the data
		copy(value, v)
		return nil
	})
	if err == nil {
		return value, nil
	}
	if errors.Cause(err) == ErrBucketNotExist || errors.Cause(err) == ErrNotExist {
		return nil, err
	}
	return nil, errors.Wrap(ErrIO, err.Error())
}

// Put inserts a <key, value> record
func (b *boltDB) Put(namespace string, key, value []byte) (err error) {
	for c := 0; c < 3; c++ {
		if err = b.db.Update(func(tx *bolt.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists([]byte(namespace))
			if err != nil {
				return err
			}
			return bucket.Put(key, value)
		}); err == nil {
			break
		}
	}
	if err != nil {
		err = errors.Wrap(ErrIO, err.Error())
	}
	return err
}

// Delete deletes a record,if key is nil,this will delete the whole bucket
func (b *boltDB) Delete(namespace string, key []byte) (err error) {
	for c := 0; c < 3; c++ {
		if key == nil {
			err = b.db.Update(func(tx *bolt.Tx) error {
				if err := tx.DeleteBucket([]byte(namespace)); err != bolt.ErrBucketNotFound {
					return err
				}
				return nil
			})
		} else {
			err = b.db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket([]byte(namespace))
				if bucket == nil {
					return nil
				}
				return bucket.Delete(key)
			})
		}
		if err == nil {
			break
		}
	}
	if err != nil {
		err = errors.Wrap(ErrIO, err.Error())
	}
	return err
}
