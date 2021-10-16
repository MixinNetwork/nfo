package mtg

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/fox-one/mixin-sdk-go"
)

type Group struct {
	mixin   *mixin.Client
	store   Store
	workers []Worker

	members   []string
	epoch     time.Time
	threshold int
	pin       string
}

func BuildGroup(ctx context.Context, store Store, conf *Configuration) (*Group, error) {
	if cg := conf.Genesis; len(cg.Members) < cg.Threshold || cg.Threshold < 1 {
		return nil, fmt.Errorf("invalid group threshold %d %d", len(cg.Members), cg.Threshold)
	}
	if !strings.Contains(strings.Join(conf.Genesis.Members, ","), conf.App.ClientId) {
		return nil, fmt.Errorf("app %s not belongs to the group", conf.App.ClientId)
	}

	s := &mixin.Keystore{
		ClientID:   conf.App.ClientId,
		SessionID:  conf.App.SessionId,
		PrivateKey: conf.App.PrivateKey,
		PinToken:   conf.App.PinToken,
	}
	client, err := mixin.NewFromKeystore(s)
	if err != nil {
		return nil, err
	}
	err = client.VerifyPin(ctx, conf.App.PIN)
	if err != nil {
		return nil, err
	}

	grp := &Group{
		mixin: client,
		store: store,
		pin:   conf.App.PIN,
	}

	for _, id := range conf.Genesis.Members {
		ts := time.Unix(0, conf.Genesis.Timestamp)
		err = grp.AddNode(id, conf.Genesis.Threshold, ts)
		if err != nil {
			return nil, err
		}
	}
	members, threshold, epoch, err := grp.ListActiveNodes()
	if err != nil {
		return nil, err
	}
	grp.members = members
	grp.threshold = threshold
	grp.epoch = epoch
	return grp, nil
}

func (grp *Group) AddWorker(wkr Worker) {
	grp.workers = append(grp.workers, wkr)
}

func (grp *Group) Run(ctx context.Context) {
	for {
		grp.drainOutputs(ctx, 100)
		grp.handleUnspentOutputs(ctx)
		grp.signTransactions(ctx)
		grp.publishTransactions(ctx)
	}
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

func (grp *Group) handleUnspentOutputs(ctx context.Context) error {
	outputs, err := grp.store.ListOutputs(mixin.UTXOStateUnspent, 16)
	if err != nil {
		return err
	}
	for _, out := range outputs {
		for _, wkr := range grp.workers {
			wkr.ProcessOutput(ctx, out)
		}
	}
	return nil
}

func (grp *Group) signTransactions(ctx context.Context) error {
	txs, err := grp.store.ListTransactions(TransactionStateInitial, 1)
	if err != nil || len(txs) != 1 {
		return err
	}
	tx := txs[0]
	raw, err := grp.signTransaction(ctx, tx)
	if err != nil {
		return err
	}
	tx.Raw = raw
	tx.UpdatedAt = time.Now()
	return grp.store.WriteTransaction(tx.TraceId, tx)
}

func (grp *Group) publishTransactions(ctx context.Context) error {
	txs, err := grp.store.ListTransactions(TransactionStateDone, 0)
	if err != nil || len(txs) == 0 {
		return err
	}
	for _, tx := range txs {
		raw := hex.EncodeToString(tx.Raw)
		h, err := grp.mixin.SendRawTransaction(ctx, raw)
		if err != nil {
			return err
		}
		s, err := grp.mixin.GetRawTransaction(ctx, *h)
		if err != nil {
			return err
		}
		if s.Snapshot == nil || !s.Snapshot.HasValue() {
			continue
		}
		err = grp.store.DeleteTransaction(tx.TraceId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (grp *Group) compactOutputs(ctx context.Context) {
	panic(0)
}
