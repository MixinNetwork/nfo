package store

import (
	"encoding/binary"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixIterationPayload = "ITERATION:PAYLOAD:"
	prefixIterationQueue   = "ITERATION:QUEUE:"
)

func (bs *BadgerStore) WriteIteration(ir *mtg.Iteration) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		old, err := bs.readIteration(txn, ir.NodeId)
		if err != nil {
			return err
		}
		if old != nil && old.Action >= ir.Action {
			return nil
		}
		if old != nil {
			key := buildIterationTimedKey(old)
			err = txn.Delete(key)
			if err != nil {
				return err
			}
			key = append([]byte(prefixIterationPayload), old.NodeId...)
			err = txn.Delete(key)
			if err != nil {
				return err
			}
		}
		key := buildIterationTimedKey(ir)
		err = txn.Set(key, []byte{1})
		if err != nil {
			return err
		}
		val := common.MsgpackMarshalPanic(ir)
		key = append([]byte(prefixIterationPayload), ir.NodeId...)
		return txn.Set(key, val)
	})
}

func (bs *BadgerStore) ListIterations() ([]*mtg.Iteration, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefixIterationQueue)
	it := txn.NewIterator(opts)
	defer it.Close()

	var irs []*mtg.Iteration
	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		key := it.Item().Key()
		id := string(key[len(opts.Prefix)+8:])
		ir, err := bs.readIteration(txn, id)
		if err != nil {
			return nil, err
		}
		irs = append(irs, ir)
	}
	return irs, nil
}

func (bs *BadgerStore) readIteration(txn *badger.Txn, id string) (*mtg.Iteration, error) {
	key := append([]byte(prefixIterationPayload), id...)
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
	var ir mtg.Iteration
	err = common.MsgpackUnmarshal(val, &ir)
	return &ir, err
}

func buildIterationTimedKey(ir *mtg.Iteration) []byte {
	buf := make([]byte, 8)
	ts := ir.CreatedAt.UnixNano()
	binary.BigEndian.PutUint64(buf, uint64(ts))
	key := append([]byte(prefixIterationQueue), buf...)
	return append(key, ir.NodeId...)
}
