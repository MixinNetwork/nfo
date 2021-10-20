package nft

import (
	"bytes"
	"context"
	"encoding/base64"

	"github.com/MixinNetwork/nfo/mtg"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
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
	min, err := decimal.NewFromString(MintMinimumCost)
	if err != nil {
		return
	}
	if out.AssetID != MintAssetId {
		return
	}
	if out.Amount.Cmp(min) < 0 {
		return
	}
	if uuid.FromStringOrNil(out.Sender).String() == uuid.Nil.String() {
		return
	}
	extra, err := base64.RawURLEncoding.DecodeString(out.Memo)
	if err != nil {
		return
	}
	nfm, err := DecodeNFOMemo(extra)
	if err != nil {
		return
	}
	if bytes.Compare(nfm.Group, NMDefaultGroupKey) == 0 {
	}
}
