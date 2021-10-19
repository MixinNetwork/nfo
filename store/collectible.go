package store

import "github.com/MixinNetwork/nfo/mtg"

func (bs *BadgerStore) WriteCollectibleOutput(out *mtg.CollectibleOutput, traceId string) error {
	panic(0)
}

func (bs *BadgerStore) ListCollectibleOutputsForTransaction(traceId string) ([]*mtg.CollectibleOutput, error) {
	panic(0)
}

func (bs *BadgerStore) WriteCollectibleTransaction(traceId string, tx *mtg.Transaction) error {
	panic(0)
}

func (bs *BadgerStore) ReadCollectibleTransaction(traceId string) (*mtg.Transaction, error) {
	panic(0)
}

func (bs *BadgerStore) ListCollectibleTransactions(state int, limit int) ([]*mtg.Transaction, error) {
	return nil, nil
}
