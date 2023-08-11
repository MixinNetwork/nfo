package store

import (
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/trusted-group/mtg"
	"github.com/dgraph-io/badger/v4"
)

const (
	prefixCollectibleOutputPayload     = "COLLECTIBLES:OUTPUT:PAYLOAD:"
	prefixCollectibleOutputState       = "COLLECTIBLES:OUTPUT:STATE:"
	prefixCollectibleOutputTransaction = "COLLECTIBLES:OUTPUT:TRASACTION:"
	prefixCollectibleOutputToken       = "COLLECTIBLES:OUTPUT:ASSET:"

	prefixCollectibleTransactionPayload = "COLLECTIBLES:TRANSACTION:PAYLOAD:"
	prefixCollectibleTransactionState   = "COLLECTIBLES:TRANSACTION:STATE:"
	prefixCollectibleTransactionHash    = "COLLECTIBLES:TRANSACTION:HASH:"
)

func (bs *BadgerStore) WriteCollectibleOutput(out *mtg.CollectibleOutput, traceId string) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		return bs.writeCollectibleOutput(txn, out, traceId)
	})
}

func (bs *BadgerStore) WriteCollectibleOutputs(outs []*mtg.CollectibleOutput, traceId string) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		for _, out := range outs {
			err := bs.writeCollectibleOutput(txn, out, traceId)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (bs *BadgerStore) ListCollectibleOutputsForTransaction(traceId string) ([]*mtg.CollectibleOutput, error) {
	prefix := prefixCollectibleOutputTransaction + traceId
	return bs.listCollectibleOutputs(prefix, 0)
}

func (bs *BadgerStore) ListCollectibleOutputsForToken(state, tokenId string, limit int) ([]*mtg.CollectibleOutput, error) {
	prefix := prefixCollectibleOutputToken + state + tokenId
	return bs.listCollectibleOutputs(prefix, limit)
}

func (bs *BadgerStore) WriteCollectibleTransaction(traceId string, tx *mtg.CollectibleTransaction) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		old, err := bs.resetOldCollectibleTransaction(txn, tx)
		if err != nil || old != nil {
			return err
		}
		key := []byte(prefixCollectibleTransactionPayload + tx.TraceId)
		val := mtg.MsgpackMarshalPanic(tx)
		err = txn.Set(key, val)
		if err != nil {
			return err
		}

		if len(tx.Raw) > 0 {
			if !tx.Hash.HasValue() {
				panic(tx.TraceId)
			}
			key = append([]byte(prefixCollectibleTransactionHash), tx.Hash[:]...)
			val = []byte(tx.TraceId)
			err = txn.Set(key, val)
			if err != nil {
				return err
			}
		}

		key = buildCollectibleTransactionTimedKey(tx)
		return txn.Set(key, []byte{1})
	})
}

func (bs *BadgerStore) ReadCollectibleTransaction(traceId string) (*mtg.CollectibleTransaction, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	return bs.readCollectibleTransaction(txn, traceId)
}

func (bs *BadgerStore) ReadCollectibleTransactionByHash(hash crypto.Hash) (*mtg.CollectibleTransaction, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	key := append([]byte(prefixCollectibleTransactionHash), hash[:]...)
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

	return bs.readCollectibleTransaction(txn, string(traceId))
}

func (bs *BadgerStore) ListCollectibleTransactions(state int, limit int) ([]*mtg.CollectibleTransaction, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(collectibleTransactionStatePrefix(state))
	it := txn.NewIterator(opts)
	defer it.Close()

	var txs []*mtg.CollectibleTransaction
	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		key := it.Item().Key()
		id := string(key[len(opts.Prefix)+8:])
		tx, err := bs.readCollectibleTransaction(txn, id)
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

func (bs *BadgerStore) listCollectibleOutputs(prefix string, limit int) ([]*mtg.CollectibleOutput, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefix)
	it := txn.NewIterator(opts)
	defer it.Close()

	var outputs []*mtg.CollectibleOutput
	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		key := it.Item().Key()
		id := string(key[len(opts.Prefix)+8:])
		out, err := bs.readCollectibleOutput(txn, id)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, out)
		if len(outputs) == limit {
			break
		}
	}
	return outputs, nil
}

func (bs *BadgerStore) writeCollectibleOutput(txn *badger.Txn, utxo *mtg.CollectibleOutput, traceId string) error {
	old, err := bs.resetOldCollectibleOutput(txn, utxo, traceId)
	if err != nil || old != nil {
		return err
	}

	key := []byte(prefixCollectibleOutputPayload + utxo.OutputId)
	val := mtg.MsgpackMarshalPanic(utxo)
	err = txn.Set(key, val)
	if err != nil {
		return err
	}

	key = buildCollectibleOutputTimedKey(utxo, prefixCollectibleOutputState, traceId)
	err = txn.Set(key, []byte{1})
	if err != nil {
		return err
	}

	key = buildCollectibleOutputTimedKey(utxo, prefixCollectibleOutputToken, traceId)
	err = txn.Set(key, []byte{1})
	if err != nil {
		return err
	}

	if traceId == "" {
		return nil
	}
	key = buildCollectibleOutputTimedKey(utxo, prefixCollectibleOutputTransaction, traceId)
	return txn.Set(key, []byte{1})
}

func (bs *BadgerStore) resetOldCollectibleOutput(txn *badger.Txn, utxo *mtg.CollectibleOutput, traceId string) (*mtg.CollectibleOutput, error) {
	old, err := bs.readCollectibleOutput(txn, utxo.OutputId)
	if err != nil || old == nil {
		return old, err
	}
	if old.State == utxo.State {
		return old, nil
	}
	if old.State > utxo.State {
		panic(old.State)
	}
	if old.SignedBy != "" && old.SignedBy != utxo.SignedBy {
		panic(old.SignedBy)
	}

	key := buildCollectibleOutputTimedKey(old, prefixCollectibleOutputState, traceId)
	err = txn.Delete(key)
	if err != nil {
		return nil, err
	}

	key = buildCollectibleOutputTimedKey(old, prefixCollectibleOutputToken, traceId)
	err = txn.Delete(key)
	if err != nil {
		return nil, err
	}

	key = buildCollectibleOutputTimedKey(old, prefixCollectibleOutputTransaction, traceId)
	return nil, txn.Delete(key)
}

func (bs *BadgerStore) readCollectibleOutput(txn *badger.Txn, id string) (*mtg.CollectibleOutput, error) {
	key := []byte(prefixCollectibleOutputPayload + id)
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
	var utxo mtg.CollectibleOutput
	err = mtg.MsgpackUnmarshal(val, &utxo)
	return &utxo, err
}

func buildCollectibleOutputTimedKey(out *mtg.CollectibleOutput, prefix string, traceId string) []byte {
	buf := tsToBytes(out.CreatedAt)
	switch prefix {
	case prefixCollectibleOutputState:
		prefix = prefix + out.StateName()
	case prefixCollectibleOutputToken:
		prefix = prefix + out.StateName() + out.TokenId
	case prefixCollectibleOutputTransaction:
		prefix = prefix + traceId
	default:
		panic(prefix)
	}
	key := append([]byte(prefix), buf...)
	return append(key, []byte(out.OutputId)...)
}

func (bs *BadgerStore) readCollectibleTransaction(txn *badger.Txn, traceId string) (*mtg.CollectibleTransaction, error) {
	key := []byte(prefixCollectibleTransactionPayload + traceId)
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
	var tx mtg.CollectibleTransaction
	err = mtg.MsgpackUnmarshal(val, &tx)
	return &tx, err
}

func (bs *BadgerStore) resetOldCollectibleTransaction(txn *badger.Txn, tx *mtg.CollectibleTransaction) (*mtg.CollectibleTransaction, error) {
	old, err := bs.readCollectibleTransaction(txn, tx.TraceId)
	if err != nil || old == nil {
		return old, err
	}
	if old.State >= tx.State {
		return old, nil
	}

	key := buildCollectibleTransactionTimedKey(old)
	_, err = txn.Get(key)
	if err != nil {
		panic(key)
	}
	return nil, txn.Delete(key)
}

func buildCollectibleTransactionTimedKey(tx *mtg.CollectibleTransaction) []byte {
	buf := tsToBytes(tx.UpdatedAt)
	prefix := collectibleTransactionStatePrefix(tx.State)
	key := append([]byte(prefix), buf...)
	return append(key, []byte(tx.TraceId)...)
}

func collectibleTransactionStatePrefix(state int) string {
	prefix := prefixCollectibleTransactionState
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
