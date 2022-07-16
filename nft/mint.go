package nft

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"

	"github.com/MixinNetwork/mixin/logger"
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
	logger.Verbosef("MintWorker.ProcessOutput(%v)\n", *out)
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
	if bytes.Compare(nfm.Encode(), extra) != 0 {
		return
	}

	ck := nfm.Collection.Bytes()
	old, err := mw.store.ReadMintToken(ck, nfm.Token)
	if err != nil {
		panic(err)
	} else if old != nil {
		return
	}
	og, err := mw.store.ReadMintCollection(ck)
	if err != nil {
		panic(err)
	}
	if og != nil && og.Creator != out.Sender && bytes.Compare(ck, NMDefaultCollectionKey) != 0 {
		return
	}
	err = mw.store.WriteMintToken(ck, nfm.Token, out.Sender)
	if err != nil {
		panic(err)
	}
	err = mw.grp.BuildCollectibleTransaction(ctx, []string{out.Sender}, 1, extra, "", "")
	logger.Verbosef("MintWorker.BuildCollectibleMintTransaction(%s, %s)\n", out.Sender, hex.EncodeToString(extra))
	if err != nil {
		panic(err)
	}
}

func (mw *MintWorker) ProcessCollectibleOutput(ctx context.Context, out *mtg.CollectibleOutput) {
	logger.Verbosef("MintWorker.ProcessCollectibleOutput(%v)\n", *out)
}
