package nft

import (
	"context"

	"github.com/MixinNetwork/nfo/mtg"
)

type MintWorker struct {
	grp   *mtg.Group
	store Store
}

func NewMintWorker(grp *mtg.Group, store Store) *MintWorker {
	return &MintWorker{
		grp:   grp,
		store: store,
	}
}

func (mw *MintWorker) ProcessOutput(ctx context.Context, out *mtg.Output) {
}
