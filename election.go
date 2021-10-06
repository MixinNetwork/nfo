package main

import (
	"context"

	"github.com/fox-one/mixin-sdk-go"
)

type ElectionWorker struct {
}

func (ew *ElectionWorker) ProcessOutput(context.Context, *mixin.MultisigUTXO) {
	panic(0)
}
