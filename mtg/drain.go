package mtg

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go"
)

const (
	outputsDrainingKey            = "outputs-draining-checkpoint"
	collectibleOutputsDrainingKey = "collectible-outputs-draining-checkpoint"
)

func (grp *Group) drainOutputsFromNetwork(ctx context.Context, filter map[string]bool, batch int) {
	for {
		checkpoint, err := grp.readDrainingCheckpoint(ctx, outputsDrainingKey)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		outputs, err := grp.mixin.ReadMultisigOutputs(ctx, grp.members, uint8(grp.threshold), checkpoint, batch)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		checkpoint = grp.processMultisigOutputs(filter, checkpoint, outputs)
		grp.writeDrainingCheckpoint(ctx, outputsDrainingKey, checkpoint)
		if len(outputs) < batch/2 {
			break
		}
	}
}

func (grp *Group) processMultisigOutputs(filter map[string]bool, checkpoint time.Time, outputs []*mixin.MultisigUTXO) time.Time {
	for _, out := range outputs {
		checkpoint = out.UpdatedAt
		key := fmt.Sprintf("OUT:%s:%d", out.UTXOID, out.UpdatedAt.UnixNano())
		if filter[key] || out.UpdatedAt.Before(grp.epoch) {
			continue
		}
		filter[key] = true
		logger.Verbosef("Group.processMultisigOutputs(%s) => %s", out.UTXOID, out.SignedTx)
		ver, extra := decodeTransactionWithExtra(out.SignedTx)
		if out.SignedTx != "" && ver == nil {
			panic(out.SignedTx)
		}
		if out.State == mixin.UTXOStateUnspent {
			grp.writeOutputOrPanic(out, "", nil)
			continue
		}
		tx := &Transaction{
			GroupId: extra.G,
			TraceId: extra.T.String(),
			State:   TransactionStateInitial,
			Raw:     ver.Marshal(),
			Hash:    ver.PayloadHash(),
		}
		if ver.AggregatedSignature != nil {
			out.State = mixin.UTXOStateSpent
			tx.State = TransactionStateSigned
		}
		grp.writeOutputOrPanic(out, tx.TraceId, tx)
	}

	for _, utxo := range outputs {
		key := fmt.Sprintf("ACT:%s:%d", utxo.UTXOID, utxo.UpdatedAt.UnixNano())
		if filter[key] || utxo.UpdatedAt.Before(grp.epoch) {
			continue
		}
		filter[key] = true
		out := NewOutputFromMultisig(utxo)
		old, err := grp.store.ReadTransactionByHash(out.TransactionHash)
		if err != nil {
			panic(out.TransactionHash)
		} else if old != nil {
			continue
		}
		grp.writeAction(out, ActionStateInitial)
	}
	return checkpoint
}

func (grp *Group) writeOutputOrPanic(utxo *mixin.MultisigUTXO, traceId string, tx *Transaction) {
	out := NewOutputFromMultisig(utxo)
	p := DecodeMixinExtra(utxo.Memo)
	if p != nil && p.G != "" {
		out.GroupId = p.G
	} else if grp.grouper != nil {
		out.GroupId = grp.grouper(out)
	}
	err := grp.store.WriteOutput(out, traceId)
	if err != nil {
		panic(err)
	}
	if traceId == "" {
		return
	}
	old, err := grp.store.ReadTransactionByTraceId(traceId)
	if err != nil {
		panic(err)
	}
	if old != nil && old.State >= TransactionStateSigned {
		return
	}
	err = grp.store.WriteTransaction(tx)
	if err != nil {
		panic(err)
	}
}

func (grp *Group) drainCollectibleOutputsFromNetwork(ctx context.Context, batch int) {
	for {
		checkpoint, err := grp.readDrainingCheckpoint(ctx, collectibleOutputsDrainingKey)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		outputs, err := grp.ReadCollectibleOutputs(ctx, grp.members, uint8(grp.threshold), checkpoint, batch)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		checkpoint = grp.processCollectibleOutputs(checkpoint, outputs)
		grp.writeDrainingCheckpoint(ctx, collectibleOutputsDrainingKey, checkpoint)
		if len(outputs) < batch/2 {
			break
		}
	}
}

func (grp *Group) processCollectibleOutputs(checkpoint time.Time, outputs []*CollectibleOutput) time.Time {
	for _, out := range outputs {
		checkpoint = out.UpdatedAt
		ver := decodeCollectibleTransaction(out.SignedTx)
		if out.SignedTx != "" && ver == nil {
			panic(out.SignedTx)
		}
		if out.State == OutputStateUnspent {
			grp.writeCollectibleOutputOrPanic(out, "", nil)
			continue
		}
		tx := &CollectibleTransaction{
			TraceId: nfoTraceId(ver.Extra),
			State:   TransactionStateInitial,
			Raw:     ver.Marshal(),
			Hash:    ver.PayloadHash(),
			NFO:     ver.Extra,
		}
		if ver.AggregatedSignature != nil {
			out.State = OutputStateSpent
			tx.State = TransactionStateSigned
		}
		grp.writeCollectibleOutputOrPanic(out, tx.TraceId, tx)
	}

	for _, out := range outputs {
		old, err := grp.store.ReadCollectibleTransactionByHash(out.TransactionHash)
		if err != nil {
			panic(out.TransactionHash)
		} else if old != nil {
			continue
		}
		grp.writeCollectibleAction(out, ActionStateInitial)
	}
	return checkpoint
}

func (grp *Group) writeCollectibleOutputOrPanic(out *CollectibleOutput, traceId string, tx *CollectibleTransaction) {
	err := grp.store.WriteCollectibleOutput(out, traceId)
	if err != nil {
		panic(err)
	}
	if traceId == "" {
		return
	}
	old, err := grp.store.ReadCollectibleTransaction(traceId)
	if err != nil {
		panic(err)
	}
	if old != nil && old.State >= TransactionStateSigned {
		return
	}
	err = grp.store.WriteCollectibleTransaction(traceId, tx)
	if err != nil {
		panic(err)
	}
}

func (grp *Group) readDrainingCheckpoint(ctx context.Context, key string) (time.Time, error) {
	val, err := grp.store.ReadProperty([]byte(key))
	if err != nil || len(val) == 0 {
		return grp.epoch, nil
	}
	ts := int64(binary.BigEndian.Uint64(val))
	return time.Unix(0, ts), nil
}

func (grp *Group) writeDrainingCheckpoint(ctx context.Context, key string, ckpt time.Time) error {
	val := make([]byte, 8)
	ts := uint64(ckpt.UnixNano())
	binary.BigEndian.PutUint64(val, ts)
	return grp.store.WriteProperty([]byte(key), val)
}
