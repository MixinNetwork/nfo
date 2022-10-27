package store

import (
	"bytes"

	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/nfo/nft"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixMintCollectionPayload = "COLLECTIBLES:MINT:GROUP:"
	prefixMintTokenPayload      = "COLLECTIBLES:MINT:TOKEN:"
)

func (bs *BadgerStore) WriteMintToken(collection []byte, id []byte, user string) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		old, err := bs.readMintToken(txn, collection, id)
		if err != nil {
			return err
		} else if old != nil {
			panic(id)
		}

		og, err := bs.readMintCollection(txn, collection)
		if err != nil {
			return err
		}
		if og == nil {
			og = &nft.Collection{
				Key:         collection,
				Creator:     user,
				Circulation: 0,
			}
		}
		if og.Creator != user && bytes.Compare(collection, mtg.NMDefaultCollectionKey) != 0 {
			panic(og.Creator)
		}
		og.Circulation += 1

		key := append([]byte(prefixMintCollectionPayload), collection...)
		err = txn.Set(key, mtg.MsgpackMarshalPanic(og))
		if err != nil {
			return err
		}
		key = append([]byte(prefixMintTokenPayload), collection...)
		key = append(key, id...)
		return txn.Set(key, []byte{1})
	})
}

func (bs *BadgerStore) ReadMintCollection(collection []byte) (*nft.Collection, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	return bs.readMintCollection(txn, collection)
}

func (bs *BadgerStore) ReadMintToken(collection, token []byte) (*nft.Token, error) {
	txn := bs.db.NewTransaction(false)
	defer txn.Discard()

	return bs.readMintToken(txn, collection, token)
}

func (bs *BadgerStore) readMintCollection(txn *badger.Txn, collection []byte) (*nft.Collection, error) {
	key := append([]byte(prefixMintCollectionPayload), collection...)
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
	var g nft.Collection
	err = mtg.MsgpackUnmarshal(val, &g)
	return &g, err
}

func (bs *BadgerStore) readMintToken(txn *badger.Txn, collection, id []byte) (*nft.Token, error) {
	key := append([]byte(prefixMintTokenPayload), collection...)
	key = append(key, id...)
	_, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &nft.Token{
		Collection: collection,
		Key:        id,
	}, nil
}
