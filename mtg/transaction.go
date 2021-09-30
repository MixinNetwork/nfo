package mtg

import (
	"encoding/hex"
	"unicode/utf8"

	"github.com/MixinNetwork/mixin/common"
	"github.com/gofrs/uuid"
)

const (
	TransactionStateInitial = "initial"
	TransactionStateDone    = "done"
)

type Transaction struct {
	TraceId   string
	State     string
	AssetId   string
	Receivers []string
	Threshold int
	Amount    string
	Raw       []byte
}

type MixinExtraPack struct {
	T uuid.UUID
	M string `msgpack:",omitempty"`
}

func marshalTransation(tx *Transaction) []byte {
	return common.MsgpackMarshalPanic(tx)
}

func parseTransaction(b []byte) *Transaction {
	var tx Transaction
	err := common.MsgpackUnmarshal(b, &tx)
	if err != nil {
		panic(err)
	}
	return &tx
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
