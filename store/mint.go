package store

import (
	"bytes"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/nfo/nft"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixMintGroupPayload = "COLLECTIBLES:MINT:GROUP:"
	prefixMintTokenPayload = "COLLECTIBLES:MINT:TOKEN:"
)

func (bs *BadgerStore) WriteMintToken(group []byte, id []byte, user string) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		old, err := bs.readMintToken(txn, group, id)
		if err != nil {
			return err
		} else if old != nil {
			panic(id)
		}

		og, err := bs.readMintGroup(txn, group)
		if err != nil {
			return err
		}
		if og == nil {
			og = &nft.Group{
				Key:         group,
				Creator:     user,
				Circulation: 0,
			}
		}
		if og.Creator != user && bytes.Compare(group, nft.NMDefaultGroupKey) != 0 {
			panic(og.Creator)
		}
		og.Circulation += 1

		key := append([]byte(prefixMintGroupPayload), group...)
		err = txn.Set(key, common.MsgpackMarshalPanic(og))
		if err != nil {
			return err
		}
		key = append([]byte(prefixMintTokenPayload), group...)
		key = append(key, id...)
		return txn.Set(key, []byte{1})
	})
}

func (bs *BadgerStore) ReadMintGroup(group []byte) (*nft.Group, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	return bs.readMintGroup(txn, group)
}

func (bs *BadgerStore) ReadMintToken(group, token []byte) (*nft.Token, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	return bs.readMintToken(txn, group, token)
}

func (bs *BadgerStore) readMintGroup(txn *badger.Txn, group []byte) (*nft.Group, error) {
	key := append([]byte(prefixMintGroupPayload), group...)
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
	var g nft.Group
	err = common.MsgpackUnmarshal(val, &g)
	return &g, err
}

func (bs *BadgerStore) readMintToken(txn *badger.Txn, group, id []byte) (*nft.Token, error) {
	key := append([]byte(prefixMintTokenPayload), group...)
	key = append(key, id...)
	_, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &nft.Token{
		Group: group,
		Key:   id,
	}, nil
}
