package mtg

import (
	"context"

	"github.com/fox-one/mixin-sdk-go"
)

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

func (grp *Group) BuildTransaction(ctx context.Context, assetId string, receivers []string, threshold int, amount string, traceId string) error {
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
	}
	raw := marshalTransation(tx)
	return grp.store.WriteTransaction(traceId, raw)
}

func (grp *Group) signTransaction(ctx context.Context, tx []byte) error {
	panic(0)
}

func (grp *Group) loop(ctx context.Context) {
	for {
		grp.drainOutputs(ctx, 100)
		grp.handleUnspentOutputs(ctx)
		grp.signTransactions(ctx)
	}
}

func (grp *Group) handleUnspentOutputs(ctx context.Context) {
}

func (grp *Group) spendOutput(out *mixin.MultisigUTXO, traceId string) error {
	if out.State != mixin.UTXOStateSpent {
		panic(out)
	}
	err := grp.store.WriteOutput(out)
	if err != nil {
		return err
	}
	b, err := grp.store.ReadTransaction(traceId)
	if err != nil || b == nil {
		return err
	}
	tx := parseTransaction(b)
	if tx.State == TransactionStateDone {
		return nil
	}
	tx.State = TransactionStateDone
	return grp.store.WriteTransaction(traceId, marshalTransation(tx))
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
