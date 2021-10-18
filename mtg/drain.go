package mtg

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/fox-one/mixin-sdk-go"
)

const outputsDrainingKey = "outputs-draining-checkpoint"

func (grp *Group) drainOutputs(ctx context.Context, batch int) {
	for {
		checkpoint, err := grp.readOutputsDrainingCheckpoint(ctx)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		outputs, err := grp.mixin.ReadMultisigOutputs(ctx, grp.members, uint8(grp.threshold), checkpoint, batch)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		checkpoint = grp.processMultisigOutputs(checkpoint, outputs)

		grp.writeOutputsDrainingCheckpoint(ctx, checkpoint)
		if len(outputs) < batch/2 {
			break
		}
	}
}

func (grp *Group) processMultisigOutputs(checkpoint time.Time, outputs []*mixin.MultisigUTXO) time.Time {
	for _, out := range outputs {
		checkpoint = out.UpdatedAt
		tx, extra := decodeTransactionWithExtra(out.SignedTx)
		if out.SignedTx != "" && tx == nil {
			continue
		}
		if tx == nil {
			grp.saveOutput(out)
			continue
		}
		as := tx.AggregatedSignature
		if as != nil && len(as.Signers) >= int(out.Threshold) {
			out.State = mixin.UTXOStateSpent
			grp.spendOutput(out, extra.T.String())
			continue
		}
		out.SignedBy = ""
		out.SignedTx = ""
		out.State = mixin.UTXOStateUnspent
		grp.saveOutput(out)
	}
	return checkpoint
}

func (grp *Group) spendOutput(utxo *mixin.MultisigUTXO, traceId string) {
	out := NewOutputFromMultisig(utxo)
	if out.State != OutputStateSpent {
		panic(out)
	}
	err := grp.store.WriteOutput(out, traceId)
	if err != nil {
		panic(err)
	}
	tx, err := grp.store.ReadTransaction(traceId)
	if err != nil {
		panic(err)
	}
	if tx == nil || tx.State == TransactionStateDone {
		return
	}
	tx.State = TransactionStateDone
	err = grp.store.WriteTransaction(traceId, tx)
	if err != nil {
		panic(err)
	}
}

func (grp *Group) saveOutput(utxo *mixin.MultisigUTXO) {
	out := NewOutputFromMultisig(utxo)
	if out.State != OutputStateUnspent {
		panic(out)
	}
	old, err := grp.store.ReadOutput(out.UTXOID)
	if err != nil {
		panic(err)
	}
	if old != nil && !old.UpdatedAt.Equal(out.UpdatedAt) {
		panic(old)
	}
	err = grp.store.WriteOutput(out, "")
	if err != nil {
		panic(err)
	}
}

func (grp *Group) readOutputsDrainingCheckpoint(ctx context.Context) (time.Time, error) {
	key := []byte(outputsDrainingKey)
	val, err := grp.store.ReadProperty(key)
	if err != nil || len(val) == 0 {
		return time.Time{}, nil
	}
	ts := int64(binary.BigEndian.Uint64(val))
	return time.Unix(0, ts), nil
}

func (grp *Group) writeOutputsDrainingCheckpoint(ctx context.Context, ckpt time.Time) error {
	val := make([]byte, 8)
	key := []byte(outputsDrainingKey)
	ts := uint64(ckpt.UnixNano())
	binary.BigEndian.PutUint64(val, ts)
	return grp.store.WriteProperty(key, val)
}
