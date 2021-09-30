package main

import (
	"context"
	"flag"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/MixinNetwork/nfo/mtg"
)

func main() {
	ctx := context.Background()

	bp := flag.String("d", "~/.mixin/data", "database directory path")
	cp := flag.String("c", "~/.mixin/nfo.toml", "configuration file path")
	flag.Parse()

	if strings.HasPrefix(*cp, "~/") {
		usr, _ := user.Current()
		*cp = filepath.Join(usr.HomeDir, (*cp)[2:])
	}
	conf, err := mtg.Setup(*cp)
	if err != nil {
		panic(err)
	}

	db, err := OpenBadger(ctx, *bp)
	if err != nil {
		panic(err)
	}

	group, err := mtg.BuildGroup(ctx, db, conf)
	if err != nil {
		panic(err)
	}
	group.AddWorker(&MintWorker{})
	group.Run(ctx)
}
