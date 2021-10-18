package mtg

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
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
	Memo      string
	Raw       []byte
	UpdatedAt time.Time
}

type MixinExtraPack struct {
	T uuid.UUID
	M string `msgpack:",omitempty"`
}

func (grp *Group) BuildTransaction(ctx context.Context, assetId string, receivers []string, threshold int, amount, memo string, traceId string) error {
	old, err := grp.store.ReadTransaction(traceId)
	if err != nil || old != nil {
		return err
	}
	tx := &Transaction{
		TraceId:   traceId,
		State:     TransactionStateInitial,
		AssetId:   assetId,
		Receivers: receivers,
		Threshold: threshold,
		Amount:    amount,
		Memo:      memo,
		UpdatedAt: time.Now(),
	}
	return grp.store.WriteTransaction(traceId, tx)
}

func (grp *Group) signTransaction(ctx context.Context, tx *Transaction) ([]byte, error) {
	outputs, err := grp.store.ListOutputsForTransaction(tx.TraceId)
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		outputs, err = grp.store.ListOutputsForAsset(mixin.UTXOStateUnspent, tx.AssetId, 36)
		if err != nil || len(outputs) == 0 {
			return nil, err
		}
	}
	if outputs[0].State == OutputStateSpent {
		tx, _ := decodeTransactionWithExtra(outputs[0].SignedTx)
		return tx.Marshal(), nil
	}

	ver := common.NewTransaction(crypto.NewHash([]byte(tx.AssetId)))
	ver.Extra = encodeMixinExtra(tx.TraceId, tx.Memo)

	var total common.Integer
	for _, out := range outputs {
		total = total.Add(common.NewIntegerFromString(out.Amount.String()))
		ver.AddInput(crypto.Hash(out.TransactionHash), out.OutputIndex)
	}
	if total.Cmp(common.NewIntegerFromString(tx.Amount)) < 0 {
		return nil, fmt.Errorf("insufficient %s %s", total, tx.Amount)
	}

	keys, err := grp.mixin.BatchReadGhostKeys(ctx, []*mixin.GhostInput{{
		Receivers: tx.Receivers,
		Index:     0,
		Hint:      tx.TraceId,
	}, {
		Receivers: grp.members,
		Index:     1,
		Hint:      tx.TraceId,
	}})
	if err != nil {
		return nil, err
	}

	amount, err := decimal.NewFromString(tx.Amount)
	if err != nil {
		return nil, err
	}
	out := keys[0].DumpOutput(uint8(tx.Threshold), amount)
	ver.Outputs = append(ver.Outputs, newCommonOutput(out))

	if diff := total.Sub(common.NewIntegerFromString(tx.Amount)); diff.Sign() > 0 {
		amount, err := decimal.NewFromString(diff.String())
		if err != nil {
			return nil, err
		}
		out := keys[1].DumpOutput(uint8(grp.threshold), amount)
		ver.Outputs = append(ver.Outputs, newCommonOutput(out))
	}

	raw := hex.EncodeToString(ver.AsLatestVersion().Marshal())
	req, err := grp.mixin.CreateMultisig(ctx, mixin.MultisigActionSign, raw)
	if err != nil {
		return nil, err
	}

	for _, out := range outputs {
		out.State = OutputStateSigned
		out.SignedBy = ver.AsLatestVersion().PayloadHash().String()
		out.SignedTx = raw
	}
	err = grp.store.WriteOutputs(outputs, tx.TraceId)
	if err != nil {
		return nil, err
	}

	req, err = grp.mixin.SignMultisig(ctx, req.RequestID, grp.pin)
	if err != nil {
		return nil, err
	}
	return ver.AsLatestVersion().Marshal(), nil
}

func decodeTransactionWithExtra(s string) (*common.VersionedTransaction, *MixinExtraPack) {
	raw, err := hex.DecodeString(s)
	if err != nil {
		return nil, nil
	}
	tx, err := common.UnmarshalVersionedTransaction(raw)
	if err != nil {
		return nil, nil
	}
	var p MixinExtraPack
	err = common.MsgpackUnmarshal(tx.Extra, &p)
	if err != nil || p.T.String() == uuid.Nil.String() {
		return nil, nil
	}
	return tx, &p
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

func newCommonOutput(out *mixin.Output) *common.Output {
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
