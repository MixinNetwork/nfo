package store

import (
	"encoding/binary"
	"time"
)

func tsToBytes(ts time.Time) []byte {
	buf := make([]byte, 8)
	d := ts.UnixNano()
	binary.BigEndian.PutUint64(buf, uint64(d))
	return buf
}
