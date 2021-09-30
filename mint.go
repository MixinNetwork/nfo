package main

import (
	"context"

	"github.com/fox-one/mixin-sdk-go"
)

type MintWorker struct {
}

func (mw *MintWorker) ProcessOutput(context.Context, *mixin.MultisigUTXO) {
	panic(0)
}
