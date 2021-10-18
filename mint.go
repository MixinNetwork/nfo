package main

import (
	"context"

	"github.com/MixinNetwork/nfo/mtg"
)

type MintWorker struct {
	grp *mtg.Group
}

func (mw *MintWorker) ProcessOutput(ctx context.Context, out *mtg.Output) {
}
