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
	ReadOutput(utxoID string) (*Output, error)
	WriteOutputs(utxos []*Output, traceId string) error

	ListOutputs(state string, limit int) ([]*Output, error)
	ListOutputsForTransaction(state, traceId string) ([]*Output, error)
	ListOutputsForAsset(state, assetId string, limit int) ([]*Output, error)

	WriteTransaction(traceId string, tx *Transaction) error
	ReadTransaction(traceId string) (*Transaction, error)
	DeleteTransaction(traceId string) error
	ListTransactions(state string, limit int) ([]*Transaction, error)
}

type Worker interface {
	ProcessOutput(context.Context, *Output)
}
