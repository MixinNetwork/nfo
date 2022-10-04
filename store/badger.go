package store

import (
	"context"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/dgraph-io/badger/v3"
)

type BadgerStore struct {
	db *badger.DB
}

func OpenBadger(ctx context.Context, path string) (*BadgerStore, error) {
	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)

	go func() {
		for {
			lsm, vlog := db.Size()
			logger.Printf("Badger LSM %d VLOG %d\n", lsm, vlog)
			if lsm > 1024*1024*8 || vlog > 1024*1024*32 {
				err := db.RunValueLogGC(0.5)
				logger.Printf("Badger RunValueLogGC %v\n", err)
			}
			time.Sleep(5 * time.Minute)
		}
	}()

	return &BadgerStore{
		db: db,
	}, err
}

func (bs *BadgerStore) Close() error {
	return bs.db.Close()
}

func (bs *BadgerStore) Badger() *badger.DB {
	return bs.db
}

func (bs *BadgerStore) WriteProperty(key, val []byte) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
}

func (bs *BadgerStore) ReadProperty(key []byte) ([]byte, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return item.ValueCopy(nil)
}
