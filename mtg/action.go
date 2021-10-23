package mtg

import (
	"context"
	"time"
)

const (
	ActionStateInitial = 10
	ActionStateDone    = 11
)

type Action struct {
	UTXOID    string
	CreatedAt time.Time
	State     int
}

// actions queue is all the utxos ordered by their created time
// this queue can only be queried after all utxos are drained from api
func (grp *Group) handleActionsQueue(ctx context.Context) error {
	outputs, err := grp.store.ListActions(16)
	if err != nil {
		return err
	}
	for _, out := range outputs {
		for _, wkr := range grp.workers {
			wkr.ProcessOutput(ctx, out)
		}
		grp.writeAction(out, ActionStateDone)
	}
	return nil
}

func (grp *Group) writeAction(out *Output, state int) {
	err := grp.store.WriteAction(&Action{
		UTXOID:    out.UTXOID,
		CreatedAt: out.CreatedAt,
		State:     state,
	})
	if err != nil {
		panic(err)
	}
}
