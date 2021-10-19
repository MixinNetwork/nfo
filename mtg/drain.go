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
		ver, extra := decodeTransactionWithExtra(out.SignedTx)
		if out.SignedTx != "" && ver == nil {
			panic(out.SignedTx)
		}
		if out.State == mixin.UTXOStateUnspent {
			grp.writeOutput(out, "", nil)
			continue
		}
		tx := &Transaction{
			TraceId: extra.T.String(),
			State:   TransactionStateSigning,
			Raw:     ver.Marshal(),
		}
		as := ver.AggregatedSignature
		if as != nil && len(as.Signers) >= int(out.Threshold) {
			out.State = mixin.UTXOStateSpent
			tx.State = TransactionStateSigned
		}
		grp.writeOutput(out, tx.TraceId, tx)
	}
	return checkpoint
}

func (grp *Group) writeOutput(utxo *mixin.MultisigUTXO, traceId string, tx *Transaction) {
	out := NewOutputFromMultisig(utxo)
	err := grp.store.WriteOutput(out, traceId)
	if err != nil {
		panic(err)
	}
	if traceId == "" {
		return
	}
	old, err := grp.store.ReadTransaction(traceId)
	if err != nil {
		panic(err)
	}
	if old != nil && old.State >= TransactionStateSigned {
		return
	}
	err = grp.store.WriteTransaction(traceId, tx)
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
