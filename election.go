package main

import (
	"context"

	"github.com/MixinNetwork/trusted-group/mtg"
)

type ElectionWorker struct {
}

func (ew *ElectionWorker) ProcessOutput(context.Context, *mtg.Output) {
}
