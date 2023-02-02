package store

import (
	"github.com/MixinNetwork/trusted-group/mtg"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixOutputPayload     = "OUTPUT:PAYLOAD:"
	prefixOutputTransaction = "OUTPUT:TRASACTION:"
	prefixOutputGroupAsset  = "OUTPUT:ASSET:"
)

func (bs *BadgerStore) WriteOutput(utxo *mtg.Output, traceId string) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		return bs.writeOutput(txn, utxo, traceId)
	})
}

func (bs *BadgerStore) WriteOutputs(utxos []*mtg.Output, traceId string) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		for _, utxo := range utxos {
			err := bs.writeOutput(txn, utxo, traceId)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (bs *BadgerStore) ListOutputsForTransaction(traceId string) ([]*mtg.Output, error) {
	prefix := prefixOutputTransaction + traceId
	return bs.listOutputs(prefix, 0)
}

func (bs *BadgerStore) ListOutputsForAsset(groupId, state, assetId string, limit int) ([]*mtg.Output, error) {
	prefix := prefixOutputGroupAsset + state + assetId + groupId
	return bs.listOutputs(prefix, limit)
}

func (bs *BadgerStore) listOutputs(prefix string, limit int) ([]*mtg.Output, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefix)
	it := txn.NewIterator(opts)
	defer it.Close()

	var outputs []*mtg.Output
	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		key := it.Item().Key()
		// asset with different group id
		if len(key) != len(opts.Prefix)+36 {
			continue
		}
		id := string(key[len(opts.Prefix)+8:])
		out, err := bs.readOutput(txn, id)
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

func (bs *BadgerStore) writeOutput(txn *badger.Txn, utxo *mtg.Output, traceId string) error {
	old, err := bs.resetOldOutput(txn, utxo, traceId)
	if err != nil || old != nil {
		return err
	}

	key := []byte(prefixOutputPayload + utxo.UTXOID)
	val := mtg.MsgpackMarshalPanic(utxo)
	err = txn.Set(key, val)
	if err != nil {
		return err
	}

	key = buildOutputTimedKey(utxo, prefixOutputGroupAsset, traceId)
	err = txn.Set(key, []byte{1})
	if err != nil {
		return err
	}

	if traceId == "" {
		return nil
	}
	key = buildOutputTimedKey(utxo, prefixOutputTransaction, traceId)
	return txn.Set(key, []byte{1})
}

func (bs *BadgerStore) resetOldOutput(txn *badger.Txn, utxo *mtg.Output, traceId string) (*mtg.Output, error) {
	old, err := bs.readOutput(txn, utxo.UTXOID)
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

	key := buildOutputTimedKey(old, prefixOutputGroupAsset, traceId)
	err = txn.Delete(key)
	if err != nil {
		return nil, err
	}

	key = buildOutputTimedKey(old, prefixOutputTransaction, traceId)
	return nil, txn.Delete(key)
}

func (bs *BadgerStore) readOutput(txn *badger.Txn, id string) (*mtg.Output, error) {
	key := []byte(prefixOutputPayload + id)
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
	var utxo mtg.Output
	err = mtg.MsgpackUnmarshal(val, &utxo)
	return &utxo, err
}

func buildOutputTimedKey(out *mtg.Output, prefix string, traceId string) []byte {
	buf := tsToBytes(out.UpdatedAt)
	switch prefix {
	case prefixOutputGroupAsset:
		prefix = prefix + out.StateName() + out.AssetID + out.GroupId
	case prefixOutputTransaction:
		prefix = prefix + traceId
	default:
		panic(prefix)
	}
	key := append([]byte(prefix), buf...)
	return append(key, []byte(out.UTXOID)...)
}
