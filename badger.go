package main

import (
	"context"

	"github.com/dgraph-io/badger/v3"
	"github.com/fox-one/mixin-sdk-go"
)

type BadgerStore struct {
	db *badger.DB
}

func OpenBadger(ctx context.Context, path string) (*BadgerStore, error) {
	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)
	return &BadgerStore{
		db: db,
	}, err
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

func (bs *BadgerStore) WriteOutput(utxo *mixin.MultisigUTXO) error {
	panic(0)
}

func (bs *BadgerStore) ReadOutput(utxoID string) (*mixin.MultisigUTXO, error) {
	panic(0)
}

func (bs *BadgerStore) WriteOutputs(utxos []*mixin.MultisigUTXO) error {
	panic(0)
}

func (bs *BadgerStore) ListOutputs(state string, limit int) ([]*mixin.MultisigUTXO, error) {
	panic(0)
}

func (bs *BadgerStore) ListOutputsForTransaction(state, traceId string) ([]*mixin.MultisigUTXO, error) {
	panic(0)
}

func (bs *BadgerStore) ListOutputsForAsset(state, assetId string, limit int) ([]*mixin.MultisigUTXO, error) {
	panic(0)
}

func (bs *BadgerStore) WriteTransaction(traceId string, raw []byte) error {
	panic(0)
}

func (bs *BadgerStore) ReadTransaction(traceId string) ([]byte, error) {
	panic(0)
}

func (bs *BadgerStore) ListTransactions(state string, limit int) ([][]byte, error) {
	panic(0)
}
