package mtg

import "time"

const (
	ActionStateInitial = 10
	ActionStateDone    = 11
)

type Action struct {
	UTXOID    string
	CreatedAt time.Time
	State     int
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
