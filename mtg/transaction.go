package mtg

import (
	"encoding/hex"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mixin-sdk-go"
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
	UpdatedAt time.Time
}

type MixinExtraPack struct {
	T uuid.UUID
	M string `msgpack:",omitempty"`
}

func outputToMainnet(out *mixin.Output) *common.Output {
	cout := &common.Output{
		Type:   common.OutputTypeScript,
		Amount: common.NewIntegerFromString(out.Amount.String()),
		Script: common.Script(out.Script),
		Mask:   crypto.Key(out.Mask),
	}
	for _, k := range out.Keys {
		ck := crypto.Key(k)
		cout.Keys = append(cout.Keys, &ck)
	}
	return cout
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
	if len(b) >= common.ExtraSizeLimit {
		panic(memo)
	}
	return b
}

func decodeMixinExtra(b []byte) *MixinExtraPack {
	var p MixinExtraPack
	common.MsgpackUnmarshal(b, &p)
	return &p
}
