package mtg

import (
	"context"
	"time"

	"github.com/fox-one/mixin-sdk-go"
)

type Store interface {
	WriteOutput(utxo *mixin.MultisigUTXO) error
	ReadOutput(utxoID string) (*mixin.MultisigUTXO, error)
	ListOutputs(state string)
	WriteTransaction(traceId string, raw []byte) error
	ReadTransaction(traceId string) ([]byte, error)
}

type Worker interface {
	ProcessOutput(context.Context, *mixin.MultisigUTXO)
	ProcessCollectible()
}

type Configuration struct {
	Members   []string
	Threshold int
}

type Group struct {
	mixin   *mixin.Client
	store   Store
	workers []Worker

	members   []string
	threshold int
}

func BuildGroup(ctx context.Context, store Store) (*Group, error) {
	s := &mixin.Keystore{
		ClientID:   "",
		SessionID:  "",
		PrivateKey: "",
		PinToken:   "",
	}

	client, err := mixin.NewFromKeystore(s)
	if err != nil {
		return nil, err
	}
	grp := &Group{
		mixin: client,
		store: store,
	}
	panic(grp)
}

func (grp *Group) AddWorker(wkr Worker) {
	grp.workers = append(grp.workers, wkr)
	panic(0)
}

func (grp *Group) Run(ctx context.Context) {
	go grp.signCollectibles(ctx)
	go grp.syncCollectibles(ctx)
	go grp.signTransactions(ctx)
	go grp.compactOutputs(ctx)
	grp.loop(ctx)
}

func (grp *Group) BuildTransaction(ctx context.Context, assetId string, receivers []string, threshold int, amount string, traceId string) ([]byte, error) {
	old, err := grp.store.ReadTransaction(traceId)
	if err != nil || old != nil {
		return old, err
	}
	var raw []byte
	err = grp.store.WriteTransaction(traceId, raw)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (grp *Group) SendCollectible(ctx context.Context, tokenId string, receivers []string, threshold int, traceId string) ([]byte, error) {
	panic(0)
}

func (grp *Group) signTransaction(ctx context.Context, tx []byte) error {
	panic(0)
}

func (grp *Group) loop(ctx context.Context) {
	for {
		grp.drainOutputs(ctx, 100)
		grp.handleUnspentOutputs(ctx)
		grp.buildTransactions(ctx)
	}
}

func (grp *Group) handleUnspentOutputs(ctx context.Context) {
}

func (grp *Group) buildTransactions(ctx context.Context) {
}

func (grp *Group) drainOutputs(ctx context.Context, batch int) {
	for {
		checkpoint, err := grp.readOutputsCheckpoint(ctx)
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
		grp.writeOutputsCheckpoint(ctx, checkpoint)
		if len(outputs) < batch/2 {
			break
		}
	}
}

func (grp *Group) spendOutput(out *mixin.MultisigUTXO, traceId string) error {
	if out.State != mixin.UTXOStateSpent {
		panic(out)
	}
	panic(0)
}

func (grp *Group) saveOutput(out *mixin.MultisigUTXO) error {
	if out.State != mixin.UTXOStateUnspent {
		panic(out)
	}
	old, err := grp.store.ReadOutput(out.UTXOID)
	if err != nil {
		return err
	}
	if old != nil && old.UpdatedAt != out.UpdatedAt {
		panic(old)
	}
	return grp.store.WriteOutput(out)
}

func (grp *Group) compactOutputs(ctx context.Context) {
	panic(0)
}

func (grp *Group) signTransactions(ctx context.Context) {
	panic(0)
}

func (grp *Group) syncCollectibles(ctx context.Context) {
	panic(0)
}

func (grp *Group) signCollectibles(ctx context.Context) {
	panic(0)
}

func (grp *Group) readOutputsCheckpoint(ctx context.Context) (time.Time, error) {
	panic(0)
}

func (grp *Group) writeOutputsCheckpoint(ctx context.Context, ckpt time.Time) error {
	panic(0)
}
