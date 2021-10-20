package nft

import (
	"context"

	"github.com/MixinNetwork/nfo/mtg"
)

const (
	MintAssetId     = "c94ac88f-4671-3976-b60a-09064f1811e8"
	MintMinimumCost = "0.001"
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
