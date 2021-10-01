package mtg

import (
	"context"

	"github.com/fox-one/mixin-sdk-go"
)

type Store interface {
	WriteProperty(key, val []byte) error
	ReadProperty(key []byte) ([]byte, error)

	WriteOutput(utxo *mixin.MultisigUTXO, traceId string) error
	ReadOutput(utxoID string) (*mixin.MultisigUTXO, error)
	WriteOutputs(utxos []*mixin.MultisigUTXO, traceId string) error

	ListOutputs(state string, limit int) ([]*mixin.MultisigUTXO, error)
	ListOutputsForTransaction(state, traceId string) ([]*mixin.MultisigUTXO, error)
	ListOutputsForAsset(state, assetId string, limit int) ([]*mixin.MultisigUTXO, error)

	WriteTransaction(traceId string, tx *Transaction) error
	ReadTransaction(traceId string) (*Transaction, error)
	ListTransactions(state string, limit int) ([]*Transaction, error)
}

type Worker interface {
	ProcessOutput(context.Context, *mixin.MultisigUTXO)
}
