package main

import (
	"context"

	"github.com/MixinNetwork/nfo/mtg"
)

type ElectionWorker struct {
}

func (ew *ElectionWorker) ProcessOutput(context.Context, *mtg.Output) {
}
