package main

import (
	"context"

	"github.com/MixinNetwork/nfo/mtg"
)

func main() {
	ctx := context.Background()
	mtg.BuildGroup(ctx, nil, nil)
}
