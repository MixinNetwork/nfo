package mtg

import (
	"context"

	"github.com/fox-one/mixin-sdk-go"
)

type Store interface {
	WriteOutput(utxo *mixin.MultisigUTXO) error
	ReadOutput(utxoID string) (*mixin.MultisigUTXO, error)
	WriteOutputs(utxos []*mixin.MultisigUTXO) error
	ListOutputs(state string, limit int) ([]*mixin.MultisigUTXO, error)
	ListOutputsForTransaction(state, traceId string) ([]*mixin.MultisigUTXO, error)
	ListOutputsForAsset(state, assetId string, limit int) ([]*mixin.MultisigUTXO, error)
	WriteTransaction(traceId string, raw []byte) error
	ReadTransaction(traceId string) ([]byte, error)
	ListTransactions(state string, limit int) ([][]byte, error)
}

type Worker interface {
	ProcessOutput(context.Context, *mixin.MultisigUTXO)
	ProcessCollectible()
}
