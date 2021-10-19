package store

import (
	"encoding/binary"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixActionPayload = "ACTION:PAYLOAD:"
	prefixActionState   = "ACTION:STATE:"
)

func (bs *BadgerStore) WriteAction(act *mtg.Action) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		old, err := bs.resetOldAction(txn, act)
		if err != nil || old != nil {
			return err
		}
		key := []byte(prefixActionPayload + act.UTXOID)
		val := common.MsgpackMarshalPanic(act)
		err = txn.Set(key, val)
		if err != nil {
			return err
		}

		key = buildActionTimedKey(act)
		return txn.Set(key, []byte{1})
	})
}

func (bs *BadgerStore) ListActions(limit int) ([]*mtg.Output, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(actionStatePrefix(mtg.ActionStateInitial))
	it := txn.NewIterator(opts)
	defer it.Close()

	var outs []*mtg.Output
	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		key := it.Item().Key()
		id := string(key[len(opts.Prefix)+8:])
		out, err := bs.readOutput(txn, id)
		if err != nil {
			return nil, err
		}
		outs = append(outs, out)
		if len(outs) == limit {
			break
		}
	}
	return outs, nil
}

func (bs *BadgerStore) resetOldAction(txn *badger.Txn, act *mtg.Action) (*mtg.Action, error) {
	old, err := bs.readAction(txn, act.UTXOID)
	if err != nil || old == nil {
		return old, err
	}
	if old.State >= act.State {
		return old, nil
	}

	key := buildActionTimedKey(old)
	_, err = txn.Get(key)
	if err != nil {
		panic(key)
	}
	return nil, txn.Delete(key)
}

func (bs *BadgerStore) readAction(txn *badger.Txn, id string) (*mtg.Action, error) {
	key := []byte(prefixActionPayload + id)
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
	var tx mtg.Action
	err = common.MsgpackUnmarshal(val, &tx)
	return &tx, err
}

func buildActionTimedKey(act *mtg.Action) []byte {
	buf := make([]byte, 8)
	ts := act.CreatedAt.UnixNano()
	binary.BigEndian.PutUint64(buf, uint64(ts))
	prefix := actionStatePrefix(act.State)
	key := append([]byte(prefix), buf...)
	return append(key, []byte(act.UTXOID)...)
}

func actionStatePrefix(state int) string {
	prefix := prefixActionState
	switch state {
	case mtg.ActionStateInitial:
		return prefix + "initial"
	case mtg.ActionStateDone:
		return prefix + "doneeee"
	}
	panic(state)
}
