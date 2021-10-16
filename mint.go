package main

import (
	"context"

	"github.com/MixinNetwork/nfo/mtg"
	"github.com/fox-one/mixin-sdk-go"
)

type MintWorker struct {
	grp *mtg.Group
}

func (mw *MintWorker) ProcessOutput(ctx context.Context, out *mtg.Output) {
	receivers := []string{out.Sender}
	traceId := mixin.UniqueConversationID(out.UTXOID, "refund")
	err := mw.grp.BuildTransaction(ctx, out.AssetID, receivers, 1, out.Amount.String(), "refund", traceId)
	if err != nil {
		panic(err)
	}
}
