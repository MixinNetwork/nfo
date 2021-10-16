package store

import (
	"encoding/binary"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixOutputPayload     = "OUTPUT:PAYLOAD:"
	prefixOutputState       = "OUTPUT:STATE:"
	prefixOutputTransaction = "OUTPUT:TRASACTION:"
	prefixOutputAsset       = "OUTPUT:ASSET:"
)

func (bs *BadgerStore) WriteOutput(utxo *mtg.Output, traceId string) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		return bs.writeOutput(txn, utxo, traceId)
	})
}

func (bs *BadgerStore) ReadOutput(utxoID string) (*mtg.Output, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	return bs.readOutput(txn, utxoID)
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

func (bs *BadgerStore) ListOutputs(state string, limit int) ([]*mtg.Output, error) {
	prefix := prefixOutputState + state
	return bs.listOutputs(prefix, limit)
}

func (bs *BadgerStore) ListOutputsForTransaction(state, traceId string) ([]*mtg.Output, error) {
	prefix := prefixOutputTransaction + state + traceId
	return bs.listOutputs(prefix, 0)
}

func (bs *BadgerStore) ListOutputsForAsset(state, assetId string, limit int) ([]*mtg.Output, error) {
	prefix := prefixOutputAsset + state + assetId
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
	err := bs.resetOldOutput(txn, utxo, traceId)
	if err != nil {
		return err
	}

	key := []byte(prefixOutputPayload + utxo.UTXOID)
	val := common.MsgpackMarshalPanic(utxo)
	err = txn.Set(key, val)
	if err != nil {
		return err
	}

	key = buildOutputTimedKey(utxo, prefixOutputState, traceId)
	err = txn.Set(key, []byte{1})
	if err != nil {
		return err
	}

	key = buildOutputTimedKey(utxo, prefixOutputAsset, traceId)
	err = txn.Set(key, []byte{1})
	if err != nil {
		return err
	}

	if utxo.SignedBy == "" {
		return nil
	}
	key = buildOutputTimedKey(utxo, prefixOutputTransaction, traceId)
	return txn.Set(key, []byte{1})
}

func (bs *BadgerStore) resetOldOutput(txn *badger.Txn, utxo *mtg.Output, traceId string) error {
	old, err := bs.readOutput(txn, utxo.UTXOID)
	if err != nil || old == nil {
		return err
	}
	if old.State == utxo.State {
		return nil
	}
	if old.SignedBy != "" && old.SignedBy != utxo.SignedBy {
		panic(old.SignedBy)
	}

	key := buildOutputTimedKey(old, prefixOutputState, traceId)
	err = txn.Delete(key)
	if err != nil {
		return err
	}

	key = buildOutputTimedKey(old, prefixOutputAsset, traceId)
	err = txn.Delete(key)
	if err != nil {
		return err
	}

	if old.SignedBy == "" {
		return nil
	}
	key = buildOutputTimedKey(old, prefixOutputTransaction, traceId)
	return txn.Delete(key)
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
	err = common.MsgpackUnmarshal(val, &utxo)
	return &utxo, err
}

func buildOutputTimedKey(out *mtg.Output, prefix string, traceId string) []byte {
	buf := make([]byte, 8)
	ts := out.UpdatedAt.UnixNano()
	binary.BigEndian.PutUint64(buf, uint64(ts))
	switch prefix {
	case prefixOutputState:
		prefix = prefix + out.StateName()
	case prefixOutputAsset:
		prefix = prefix + out.StateName() + out.AssetID
	case prefixOutputTransaction:
		prefix = prefix + out.StateName() + traceId
	default:
		panic(prefix)
	}
	key := append([]byte(prefix), buf...)
	return append(key, []byte(out.UTXOID)...)
}
