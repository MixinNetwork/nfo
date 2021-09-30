package mtg

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/shopspring/decimal"
)

type Group struct {
	mixin   *mixin.Client
	store   Store
	workers []Worker

	members   []string
	threshold int
	pin       string
}

func BuildGroup(ctx context.Context, store Store, conf *Configuration) (*Group, error) {
	if cg := conf.Group; len(cg.Members) < cg.Threshold || cg.Threshold < 1 {
		return nil, fmt.Errorf("invalid group threshold %d %d", len(cg.Members), cg.Threshold)
	}
	if !strings.Contains(strings.Join(conf.Group.Members, ","), conf.App.ClientId) {
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
		mixin:     client,
		store:     store,
		members:   conf.Group.Members,
		threshold: conf.Group.Threshold,
		pin:       conf.App.PIN,
	}
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
	}
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

func (grp *Group) signTransaction(ctx context.Context, tx *Transaction) ([]byte, error) {
	outputs, err := grp.store.ListOutputsForTransaction(mixin.UTXOStateSigned, tx.TraceId)
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		outputs, err = grp.store.ListOutputsForAsset(mixin.UTXOStateUnspent, tx.AssetId, 36)
		if err != nil {
			return nil, err
		}
	}
	var total common.Integer
	ver := common.NewTransaction(crypto.NewHash([]byte(tx.AssetId)))
	for _, out := range outputs {
		total = total.Add(common.NewIntegerFromString(out.Amount.String()))
		ver.AddInput(crypto.Hash(out.TransactionHash), out.OutputIndex)
	}
	if total.Cmp(common.NewIntegerFromString(tx.Amount)) < 0 {
		return nil, fmt.Errorf("insufficient %s %s", total, tx.Amount)
	}
	inputs := []*mixin.GhostInput{{
		Receivers: tx.Receivers,
		Index:     0,
		Hint:      tx.TraceId,
	}, {
		Receivers: grp.members,
		Index:     1,
		Hint:      tx.TraceId,
	}}
	keys, err := grp.mixin.BatchReadGhostKeys(ctx, inputs)
	if err != nil {
		return nil, err
	}

	amount, err := decimal.NewFromString(tx.Amount)
	if err != nil {
		return nil, err
	}
	out := keys[0].DumpOutput(uint8(tx.Threshold), amount)
	ver.Outputs = append(ver.Outputs, outputToMainnet(out))

	if diff := total.Sub(common.NewIntegerFromString(tx.Amount)); diff.Sign() > 0 {
		amount, err := decimal.NewFromString(diff.String())
		if err != nil {
			return nil, err
		}
		out := keys[0].DumpOutput(uint8(grp.threshold), amount)
		ver.Outputs = append(ver.Outputs, outputToMainnet(out))
	}

	raw := hex.EncodeToString(ver.AsLatestVersion().Marshal())
	req, err := grp.mixin.CreateMultisig(ctx, mixin.MultisigActionSign, raw)
	if err != nil {
		return nil, err
	}

	for _, out := range outputs {
		out.State = mixin.UTXOStateSigned
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
	tx := parseTransaction(txs[0])
	raw, err := grp.signTransaction(ctx, tx)
	if err != nil {
		return err
	}
	tx.Raw = raw
	raw = marshalTransation(tx)
	return grp.store.WriteTransaction(tx.TraceId, raw)
}

func (grp *Group) spendOutput(out *mixin.MultisigUTXO, traceId string) error {
	if out.State != mixin.UTXOStateSpent {
		panic(out)
	}
	err := grp.store.WriteOutput(out, traceId)
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
	return grp.store.WriteOutput(out, "")
}

func (grp *Group) compactOutputs(ctx context.Context) {
	panic(0)
}
