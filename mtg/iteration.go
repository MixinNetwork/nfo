package mtg

import "time"

const (
	IterationActionAdd    = "ADD"
	IterationActionRemove = "REMOVE"
)

type Iteration struct {
	Action    string
	NodeId    string
	CreatedAt time.Time
}

func (grp *Group) AddNode(id string) error {
	panic(0)
}

func (grp *Group) RemoveNode(id string) error {
	panic(0)
}
