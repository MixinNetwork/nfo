package mtg

import "time"

const (
	IterationActionAdd    = "ADD"
	IterationActionRemove = "REMOVE"
)

type Iteration struct {
	Action    string
	NodeId    string
	Threshold int
	CreatedAt time.Time
}

func (grp *Group) AddNode(id string, threshold int, timestamp time.Time) error {
	panic(0)
}

func (grp *Group) RemoveNode(id string, threshold int, timestamp time.Time) error {
	panic(0)
}

func (grp *Group) ListActiveNodes() ([]string, int, time.Time, error) {
	panic(0)
}
