package store

import (
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/trusted-group/mtg"
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
		val := mtg.MsgpackMarshalPanic(tx)
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

func (bs *BadgerStore) DeleteTransaction(old *mtg.Transaction) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		err := bs.resetTransactionOutputs(txn, old.TraceId)
		if err != nil {
			return err
		}

		key := []byte(prefixTransactionPayload + old.TraceId)
		err = txn.Delete(key)
		if err != nil {
			return err
		}

		key = buildTransactionTimedKey(old)
		return txn.Delete(key)
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

	traceId, err := bs.readTransactionTraceId(txn, hash.String())
	if err != nil || traceId == "" {
		return nil, err
	}
	return bs.readTransaction(txn, traceId)
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
	err = mtg.MsgpackUnmarshal(val, &tx)
	return &tx, err
}

func (bs *BadgerStore) readTransactionTraceId(txn *badger.Txn, hash string) (string, error) {
	key := append([]byte(prefixTransactionHash), hash[:]...)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return "", nil
	} else if err != nil {
		return "", err
	}
	traceId, err := item.ValueCopy(nil)
	if err != nil {
		return "", err
	}
	return string(traceId), nil
}

func (bs *BadgerStore) resetOldTransaction(txn *badger.Txn, tx *mtg.Transaction) (*mtg.Transaction, error) {
	old, err := bs.readTransaction(txn, tx.TraceId)
	if err != nil || old == nil {
		return nil, err
	}
	switch {
	case old.State == tx.State && old.Hash == tx.Hash:
		return old, nil
	case tx.State > old.State:
	case old.State == mtg.TransactionStateSigning && tx.State == mtg.TransactionStateInitial:
		err := bs.resetTransactionOutputs(txn, tx.TraceId)
		if err != nil {
			return nil, err
		}
	case old.State > tx.State:
		panic(old.TraceId)
	case old.Raw != nil && old.Hash != tx.Hash:
		panic(old.Hash.String())
	}

	key := buildTransactionTimedKey(old)
	_, err = txn.Get(key)
	if err != nil {
		panic(key)
	}
	return nil, txn.Delete(key)
}

func (bs *BadgerStore) resetTransactionOutputs(txn *badger.Txn, traceId string) error {
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefixOutputTransaction + traceId)
	it := txn.NewIterator(opts)
	defer it.Close()

	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		key := it.Item().Key()
		// asset list may have different group id
		// prefix + (group id) + timestamp + uuid
		if len(key) != len(opts.Prefix)+8+36 {
			continue
		}
		err := txn.Delete(key)
		if err != nil {
			return err
		}
	}
	return nil
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
