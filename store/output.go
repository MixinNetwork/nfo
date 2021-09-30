package store

import (
	"encoding/binary"

	"github.com/MixinNetwork/mixin/common"
	"github.com/dgraph-io/badger/v3"
	"github.com/fox-one/mixin-sdk-go"
)

const (
	prefixOutputPayload     = "OUTPUT:PAYLOAD:"
	prefixOutputState       = "OUTPUT:STATE:"
	prefixOutputTransaction = "OUTPUT:TRASACTION:"
	prefixOutputAsset       = "OUTPUT:ASSET:"
)

func (bs *BadgerStore) WriteOutput(utxo *mixin.MultisigUTXO) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		return bs.writeOutput(txn, utxo)
	})
}

func (bs *BadgerStore) ReadOutput(utxoID string) (*mixin.MultisigUTXO, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	return bs.readOutput(txn, utxoID)
}

func (bs *BadgerStore) WriteOutputs(utxos []*mixin.MultisigUTXO) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		for _, utxo := range utxos {
			err := bs.writeOutput(txn, utxo)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (bs *BadgerStore) ListOutputs(state string, limit int) ([]*mixin.MultisigUTXO, error) {
	prefix := prefixOutputState + state
	return bs.listOutputs(prefix, limit)
}

func (bs *BadgerStore) ListOutputsForTransaction(state, traceId string) ([]*mixin.MultisigUTXO, error) {
	prefix := prefixOutputTransaction + state + traceId
	return bs.listOutputs(prefix, 0)
}

func (bs *BadgerStore) ListOutputsForAsset(state, assetId string, limit int) ([]*mixin.MultisigUTXO, error) {
	prefix := prefixOutputAsset + state + assetId
	return bs.listOutputs(prefix, limit)
}

func (bs *BadgerStore) listOutputs(prefix string, limit int) ([]*mixin.MultisigUTXO, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefix)
	it := txn.NewIterator(opts)
	defer it.Close()

	var outputs []*mixin.MultisigUTXO
	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		key := it.Item().Key()
		id := string(key[len(opts.Prefix)+8:])
		out, err := bs.readOutput(txn, id)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, out)
	}
	return outputs, nil
}

func (bs *BadgerStore) writeOutput(txn *badger.Txn, utxo *mixin.MultisigUTXO) error {
	val := common.MsgpackMarshalPanic(utxo)
	key := []byte(prefixOutputPayload + utxo.UTXOID)
	err := txn.Set(key, val)
	if err != nil {
		return err
	}

	key = buildOutputTimedKey(utxo, prefixOutputState)
	err = txn.Set(key, []byte{1})
	if err != nil {
		return err
	}

	key = buildOutputTimedKey(utxo, prefixOutputAsset)
	err = txn.Set(key, []byte{1})
	if err != nil {
		return err
	}

	if utxo.SignedBy == "" {
		return nil
	}
	key = buildOutputTimedKey(utxo, prefixOutputTransaction)
	return txn.Set(key, []byte{1})
}

func (bs *BadgerStore) readOutput(txn *badger.Txn, id string) (*mixin.MultisigUTXO, error) {
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
	var utxo mixin.MultisigUTXO
	err = common.MsgpackUnmarshal(val, &utxo)
	return &utxo, err
}

func buildOutputTimedKey(out *mixin.MultisigUTXO, prefix string) []byte {
	buf := make([]byte, 8)
	ts := out.UpdatedAt.UnixNano()
	binary.BigEndian.PutUint64(buf, uint64(ts))
	switch prefix {
	case prefixOutputState:
		prefix = prefix + out.State
	case prefixOutputAsset:
		prefix = prefix + out.State + out.AssetID
	case prefixOutputTransaction:
		prefix = prefix + out.State + out.SignedBy
		panic("should use trace id here")
	default:
		panic(prefix)
	}
	key := append([]byte(prefix), buf...)
	return append(key, []byte(out.UTXOID)...)
}
