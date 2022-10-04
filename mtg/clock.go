package mtg

import (
	"encoding/binary"
	"sync"
	"time"
)

const clockStorePropertyKey = "MTG:GROUP:CLOCK:MONOTONIC"

type Clock struct {
	sync.RWMutex
	store Store
	now   time.Time
}

func NewClock(store Store) (*Clock, error) {
	bs, err := store.ReadProperty([]byte(clockStorePropertyKey))
	if err != nil {
		return nil, err
	}
	ts := time.Unix(0, int64(binary.BigEndian.Uint64(bs)))
	if now := time.Now(); ts.Before(now) {
		ts = now
	}
	clock := new(Clock)
	clock.store = store
	clock.now = ts
	return clock, nil
}

func (c *Clock) Now() time.Time {
	c.Lock()
	defer c.Unlock()

	for {
		now := time.Now()
		if now.After(c.now) {
			c.now = now
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	val := binary.BigEndian.AppendUint64(nil, uint64(c.now.UnixNano()))
	for {
		err := c.store.WriteProperty([]byte(clockStorePropertyKey), val)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return c.now
}
