package store

import "github.com/MixinNetwork/nfo/mtg"

const (
	prefixIteration = "ITERATION:"
)

func (bs *BadgerStore) WriteIteration(ir *mtg.Iteration) error {
	panic(0)
}

func (bs *BadgerStore) ListIterations() ([]*mtg.Iteration, error) {
	panic(0)
}
