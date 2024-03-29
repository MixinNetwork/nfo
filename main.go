package main

import (
	"context"
	"flag"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mtg"
	"github.com/MixinNetwork/nfo/nft"
	"github.com/MixinNetwork/nfo/store"
)

func main() {
	logger.SetLevel(logger.VERBOSE)
	ctx := context.Background()

	bp := flag.String("d", "~/.mixin/nfo/data", "database directory path")
	cp := flag.String("c", "~/.mixin/nfo/config.toml", "configuration file path")
	flag.Parse()

	if strings.HasPrefix(*cp, "~/") {
		usr, _ := user.Current()
		*cp = filepath.Join(usr.HomeDir, (*cp)[2:])
	}
	conf, err := mtg.Setup(*cp)
	if err != nil {
		panic(err)
	}

	if strings.HasPrefix(*bp, "~/") {
		usr, _ := user.Current()
		*bp = filepath.Join(usr.HomeDir, (*bp)[2:])
	}
	db, err := store.OpenBadger(ctx, *bp)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	group, err := mtg.BuildGroup(ctx, db, conf)
	if err != nil {
		panic(err)
	}
	mw := nft.NewMintWorker(group, db)
	group.AddWorker(mw)
	rw := NewMessengerWorker(ctx, group, conf)
	group.AddWorker(rw)
	group.Run(ctx)
}
