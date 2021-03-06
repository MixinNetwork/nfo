package store

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixTransactionPayload = "TRANSACTION:PAYLOAD:"
	prefixTransactionState   = "TRANSACTION:STATE:"
	prefixTransactionHash    = "TRANSACTION:HASH:"
)

func (bs *BadgerStore) WriteTransaction(tx *mtg.Transaction) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		old, err := bs.resetOldTransaction(txn, tx)
		if err != nil || old != nil {
			return err
		}
		key := []byte(prefixTransactionPayload + tx.TraceId)
		val := common.MsgpackMarshalPanic(tx)
		err = txn.Set(key, val)
		if err != nil {
			return err
		}

		if len(tx.Raw) > 0 {
			if !tx.Hash.HasValue() {
				panic(tx.TraceId)
			}
			key = append([]byte(prefixTransactionHash), tx.Hash[:]...)
			val = []byte(tx.TraceId)
			err = txn.Set(key, val)
			if err != nil {
				return err
			}
		}

		key = buildTransactionTimedKey(tx)
		return txn.Set(key, []byte{1})
	})
}

func (bs *BadgerStore) ReadTransactionByTraceId(traceId string) (*mtg.Transaction, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	return bs.readTransaction(txn, traceId)
}

func (bs *BadgerStore) ReadTransactionByHash(hash crypto.Hash) (*mtg.Transaction, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	key := append([]byte(prefixTransactionHash), hash[:]...)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	traceId, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return bs.readTransaction(txn, string(traceId))
}

func (bs *BadgerStore) ListTransactions(state int, limit int) ([]*mtg.Transaction, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(transactionStatePrefix(state))
	it := txn.NewIterator(opts)
	defer it.Close()

	var txs []*mtg.Transaction
	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		key := it.Item().Key()
		id := string(key[len(opts.Prefix)+8:])
		tx, err := bs.readTransaction(txn, id)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
		if len(txs) == limit {
			break
		}
	}
	return txs, nil
}

func (bs *BadgerStore) readTransaction(txn *badger.Txn, traceId string) (*mtg.Transaction, error) {
	key := []byte(prefixTransactionPayload + traceId)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	val, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	var tx mtg.Transaction
	err = common.MsgpackUnmarshal(val, &tx)
	return &tx, err
}

func (bs *BadgerStore) resetOldTransaction(txn *badger.Txn, tx *mtg.Transaction) (*mtg.Transaction, error) {
	old, err := bs.readTransaction(txn, tx.TraceId)
	if err != nil || old == nil {
		return old, err
	}
	if old.State >= tx.State {
		return old, nil
	}

	key := buildTransactionTimedKey(old)
	_, err = txn.Get(key)
	if err != nil {
		panic(key)
	}
	return nil, txn.Delete(key)
}

func buildTransactionTimedKey(tx *mtg.Transaction) []byte {
	buf := tsToBytes(tx.UpdatedAt)
	prefix := transactionStatePrefix(tx.State)
	key := append([]byte(prefix), buf...)
	return append(key, []byte(tx.TraceId)...)
}

func transactionStatePrefix(state int) string {
	prefix := prefixTransactionState
	switch state {
	case mtg.TransactionStateInitial:
		return prefix + "initiall"
	case mtg.TransactionStateSigning:
		return prefix + "signingg"
	case mtg.TransactionStateSigned:
		return prefix + "signeddd"
	case mtg.TransactionStateSnapshot:
		return prefix + "snapshot"
	}
	panic(state)
}
