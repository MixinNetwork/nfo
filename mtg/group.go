package mtg

import (
	"context"
)

type Store interface {
	WriteOutput()
	ReadOutput()
	ListOutputs()
}

type Worker interface {
	ProcessOutput()
}

type Configuration struct {
	Members   []string
	Threshold int
}

type Group struct {
}

func BuildGroup(ctx context.Context, store Store, worker Worker) (*Group, error) {
	panic(0)
}

func (grp *Group) Run(ctx context.Context) {
	go grp.signCollectibles(ctx)
	go grp.syncCollectibles(ctx)
	go grp.signTransactions(ctx)
	go grp.compactOutputs(ctx)
	grp.syncOutputs(ctx)
}

func (grp *Group) BuildTransaction(ctx context.Context, assetId string, receivers []string, threshold int, amount string) ([]byte, error) {
	panic(0)
}

func (grp *Group) SendCollectible(ctx context.Context, tokenId string, receivers []string, threshold int) ([]byte, error) {
	panic(0)
}

func (grp *Group) signTransaction(ctx context.Context, tx []byte) error {
	panic(0)
}

func (grp *Group) syncOutputs(ctx context.Context) {
	panic(0)
}

func (grp *Group) compactOutputs(ctx context.Context) {
	panic(0)
}

func (grp *Group) signTransactions(ctx context.Context) {
	panic(0)
}

func (grp *Group) syncCollectibles(ctx context.Context) {
	panic(0)
}

func (grp *Group) signCollectibles(ctx context.Context) {
	panic(0)
}
