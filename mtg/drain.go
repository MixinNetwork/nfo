package mtg

import (
	"context"
	"time"

	"github.com/fox-one/mixin-sdk-go"
)

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
				_, extra := decodeTransactionOrPanic(out.SignedTx)
				err = grp.spendOutput(out, extra.T.String())
			case mixin.UTXOStateSigned:
				tx, extra := decodeTransactionOrPanic(out.SignedTx)
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
	panic(0)
}

func (grp *Group) writeOutputsDrainingCheckpoint(ctx context.Context, ckpt time.Time) error {
	panic(0)
}
