package mtg

import (
	"encoding/hex"
	"unicode/utf8"

	"github.com/MixinNetwork/mixin/common"
	"github.com/gofrs/uuid"
)

type MixinExtraPack struct {
	T uuid.UUID
	M string `msgpack:",omitempty"`
}

func decodeTransactionOrPanic(s string) (*common.VersionedTransaction, *MixinExtraPack) {
	raw, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	tx, err := common.UnmarshalVersionedTransaction(raw)
	if err != nil {
		panic(err)
	}
	extra := decodeMixinExtra(tx.Extra)
	if extra.T.String() == uuid.Nil.String() {
		return nil, nil
	}
	return tx, extra
}

func encodeMixinExtra(traceId, memo string) []byte {
	id, err := uuid.FromString(traceId)
	if err != nil {
		panic(err)
	}
	p := &MixinExtraPack{T: id, M: memo}
	b := common.MsgpackMarshalPanic(p)
	if len(b) < common.ExtraSizeLimit {
		return b
	}
	p.M = ""
	return common.MsgpackMarshalPanic(p)
}

func decodeMixinExtra(b []byte) *MixinExtraPack {
	var p MixinExtraPack
	err := common.MsgpackUnmarshal(b, &p)
	if err == nil && (p.M != "" || p.T.String() != uuid.Nil.String()) {
		return &p
	}

	if utf8.Valid(b) {
		p.M = string(b)
	} else {
		p.M = hex.EncodeToString(b)
	}
	return &p
}
