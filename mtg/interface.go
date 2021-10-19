package mtg

import (
	"context"
)

type Store interface {
	WriteProperty(key, val []byte) error
	ReadProperty(key []byte) ([]byte, error)

	WriteIteration(ir *Iteration) error
	ListIterations() ([]*Iteration, error)

	WriteOutput(utxo *Output, traceId string) error
	WriteOutputs(utxos []*Output, traceId string) error

	ListOutputs(state string, limit int) ([]*Output, error)
	ListOutputsForTransaction(traceId string) ([]*Output, error)
	ListOutputsForAsset(state, assetId string, limit int) ([]*Output, error)

	WriteAction(act *Action) error
	ListActions(limit int) ([]*Output, error)

	WriteTransaction(traceId string, tx *Transaction) error
	ReadTransaction(traceId string) (*Transaction, error)
	ListTransactions(state int, limit int) ([]*Transaction, error)

	WriteCollectibleOutput(utxo *CollectibleOutput, traceId string) error
	ListCollectibleOutputsForTransaction(traceId string) ([]*CollectibleOutput, error)

	WriteCollectibleTransaction(traceId string, tx *Transaction) error
	ReadCollectibleTransaction(traceId string) (*Transaction, error)
	ListCollectibleTransactions(state int, limit int) ([]*Transaction, error)
}

type Worker interface {
	ProcessOutput(context.Context, *Output)
}
