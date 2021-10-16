package main

import (
	"context"
	"flag"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/nfo/store"
)

func main() {
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
	group.AddWorker(&ElectionWorker{})
	group.AddWorker(&MintWorker{})
	group.Run(ctx)
}
