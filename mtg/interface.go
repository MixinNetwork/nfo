package mtg

import (
	"context"

	"github.com/fox-one/mixin-sdk-go"
)

type Store interface {
	WriteOutput(utxo *mixin.MultisigUTXO) error
	ReadOutput(utxoID string) (*mixin.MultisigUTXO, error)
	ListOutputs(state string)
	WriteTransaction(traceId string, raw []byte) error
	ReadTransaction(traceId string) ([]byte, error)
}

type Worker interface {
	ProcessOutput(context.Context, *mixin.MultisigUTXO)
	ProcessCollectible()
}
