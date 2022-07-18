package mtg

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go"
)

const (
	outputsDrainingKey = "outputs-draining-checkpoint"
)

func (grp *Group) drainOutputsFromNetwork(ctx context.Context, filter map[string]bool, batch int) {
	for {
		checkpoint, err := grp.readDrainingCheckpoint(ctx, outputsDrainingKey)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		outputs, err := grp.ReadUnifiedOutputs(ctx, grp.members, uint8(grp.threshold), checkpoint, batch)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		checkpoint = grp.processUnifiedOutputs(filter, checkpoint, outputs)
		grp.writeDrainingCheckpoint(ctx, outputsDrainingKey, checkpoint)
		if len(outputs) < batch/2 {
			break
		}
	}
}

func (grp *Group) processUnifiedOutputs(filter map[string]bool, checkpoint time.Time, outputs []*UnifiedOutput) time.Time {
	for _, out := range outputs {
		checkpoint = out.UpdatedAt
		key := fmt.Sprintf("OUT:%s:%d", out.UniqueId(), out.UpdatedAt.UnixNano())
		if filter[key] || out.UpdatedAt.Before(grp.epoch) {
			continue
		}
		filter[key] = true
		logger.Verbosef("Group.processUnifiedOutputs(%s, %s) => %s", out.Type, out.UniqueId(), out.SignedTx)
		if out.Type == OutputTypeMultisig {
			grp.processMultisigOutput(out.AsMultisig())
		} else if out.Type == OutputTypeCollectible {
			grp.processCollectibleOutput(out.AsCollectible())
		}
	}

	for _, utxo := range outputs {
		key := fmt.Sprintf("ACT:%s:%d", utxo.UniqueId(), utxo.UpdatedAt.UnixNano())
		if filter[key] || utxo.UpdatedAt.Before(grp.epoch) {
			continue
		}
		filter[key] = true
		old, err := grp.readOldTransaction(utxo)
		if err != nil {
			panic(utxo.TransactionHash)
		} else if old != nil {
			continue
		}
		grp.writeAction(utxo, ActionStateInitial)
	}
	return checkpoint
}

func (grp *Group) readOldTransaction(utxo *UnifiedOutput) (interface{}, error) {
	if utxo.Type == OutputTypeMultisig {
		return grp.store.ReadTransactionByHash(utxo.TransactionHash)
	} else if utxo.Type == OutputTypeCollectible {
		return grp.store.ReadCollectibleTransactionByHash(utxo.TransactionHash)
	}
	panic(utxo.Type)
}

func (grp *Group) processMultisigOutput(out *Output) {
	ver, extra := decodeTransactionWithExtra(out.SignedTx)
	if out.SignedTx != "" && ver == nil {
		panic(out.SignedTx)
	}
	if out.State == OutputStateUnspent {
		grp.writeOutputOrPanic(out, "", nil)
		return
	}
	tx := &Transaction{
		GroupId: extra.G,
		TraceId: extra.T.String(),
		State:   TransactionStateInitial,
		Raw:     ver.Marshal(),
		Hash:    ver.PayloadHash(),
	}
	if ver.AggregatedSignature != nil {
		out.State = OutputStateSpent
		tx.State = TransactionStateSigned
	}
	grp.writeOutputOrPanic(out, tx.TraceId, tx)
}

func (grp *Group) writeOutputOrPanic(out *Output, traceId string, tx *Transaction) {
	p := DecodeMixinExtra(out.Memo)
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

func (grp *Group) processCollectibleOutput(out *CollectibleOutput) {
	ver := decodeCollectibleTransaction(out.SignedTx)
	if out.SignedTx != "" && ver == nil {
		panic(out.SignedTx)
	}
	if out.State == OutputStateUnspent {
		grp.writeCollectibleOutputOrPanic(out, "", nil)
		return
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
		return grp.epoch, err
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

func (grp *Group) ReadUnifiedOutputs(ctx context.Context, members []string, threshold uint8, offset time.Time, limit int) ([]*UnifiedOutput, error) {
	params := make(map[string]string)
	if !offset.IsZero() {
		params["offset"] = offset.UTC().Format(time.RFC3339Nano)
	}
	if limit > 0 {
		params["limit"] = fmt.Sprint(limit)
	}
	if threshold < 1 || int(threshold) >= len(members) {
		return nil, errors.New("invalid members")
	}
	params["members"] = mixin.HashMembers(members)
	params["threshold"] = fmt.Sprint(threshold)

	var outputs []*UnifiedOutput
	err := grp.mixin.Get(ctx, "/outputs", params, &outputs)
	if err != nil {
		return nil, err
	}
	return outputs, nil
}
