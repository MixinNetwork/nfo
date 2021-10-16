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

		for _, out := range outputs {
			switch out.State {
			case mixin.UTXOStateSpent:
				_, extra := decodeTransactionWithExtra(out.SignedTx)
				err = grp.spendOutput(out, extra.T.String())
			case mixin.UTXOStateSigned:
				tx, extra := decodeTransactionWithExtra(out.SignedTx)
				as := tx.AggregatedSignature
				if as != nil && len(as.Signers) >= int(out.Threshold) {
					out.State = mixin.UTXOStateSpent
					err = grp.spendOutput(out, extra.T.String())
				} else {
					out.SignedBy = ""
					out.SignedTx = ""
					out.State = mixin.UTXOStateUnspent
					err = grp.saveOutput(out)
				}
			case mixin.UTXOStateUnspent:
				err = grp.saveOutput(out)
			}
			if err != nil {
				break
			}
			checkpoint = out.UpdatedAt
		}

		grp.writeOutputsDrainingCheckpoint(ctx, checkpoint)
		if len(outputs) < batch/2 {
			break
		}
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
