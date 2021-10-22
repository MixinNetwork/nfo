package mtg

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
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
	go grp.loopCollectibles(ctx)
	grp.loopMultsigis(ctx)
}

func (grp *Group) loopCollectibles(ctx context.Context) {
	for {
		grp.drainCollectibleOutputsFromNetwork(ctx, 100)
		grp.signCollectibleTransactions(ctx)
		grp.publishCollectibleTransactions(ctx)
	}
}

func (grp *Group) loopMultsigis(ctx context.Context) {
	for {
		grp.drainOutputsFromNetwork(ctx, 100)
		grp.handleActionsQueue(ctx)
		grp.signTransactions(ctx)
		grp.publishTransactions(ctx)
	}
}

func (grp *Group) GetMembers() []string {
	return grp.members
}

func (grp *Group) GetThreshold() int {
	return grp.threshold
}

func (grp *Group) handleActionsQueue(ctx context.Context) error {
	outputs, err := grp.store.ListActions(16)
	if err != nil {
		return err
	}
	for _, out := range outputs {
		for _, wkr := range grp.workers {
			wkr.ProcessOutput(ctx, out)
		}
		grp.writeAction(out, ActionStateDone)
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
	logger.Verbosef("Group.signTransaction(%v) => %s %v", *tx, hex.EncodeToString(raw), err)
	if err != nil {
		return err
	}
	tx.Raw = raw
	tx.UpdatedAt = time.Now()
	tx.State = TransactionStateSigning

	ver, _ := common.UnmarshalVersionedTransaction(raw)
	extra, _ := base64.RawURLEncoding.DecodeString(string(ver.Extra))
	var p MixinExtraPack
	err = common.MsgpackUnmarshal(extra, &p)
	if p.T.String() != tx.TraceId {
		panic(hex.EncodeToString(raw))
	}

	return grp.store.WriteTransaction(tx.TraceId, tx)
}

func (grp *Group) publishTransactions(ctx context.Context) error {
	txs, err := grp.store.ListTransactions(TransactionStateSigned, 0)
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
		tx.State = TransactionStateSnapshot
		err = grp.store.WriteTransaction(tx.TraceId, tx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (grp *Group) compactOutputs(ctx context.Context) {
	panic(0)
}

func (grp *Group) signCollectibleTransactions(ctx context.Context) error {
	txs, err := grp.store.ListCollectibleTransactions(TransactionStateInitial, 1)
	if err != nil || len(txs) != 1 {
		return err
	}
	tx := txs[0]
	raw, err := grp.signCollectibleMintTransaction(ctx, tx)
	logger.Verbosef("Group.signCollectibleTransaction(%v) => %s %v", *tx, hex.EncodeToString(raw), err)
	if err != nil {
		return err
	}
	tx.Raw = raw
	tx.UpdatedAt = time.Now()
	tx.State = TransactionStateSigning

	ver, _ := common.UnmarshalVersionedTransaction(raw)
	if nfoTraceId(ver.Extra) != tx.TraceId {
		panic(hex.EncodeToString(raw))
	}

	return grp.store.WriteCollectibleTransaction(tx.TraceId, tx)
}

func (grp *Group) publishCollectibleTransactions(ctx context.Context) error {
	txs, err := grp.store.ListCollectibleTransactions(TransactionStateSigned, 0)
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
		tx.State = TransactionStateSnapshot
		err = grp.store.WriteCollectibleTransaction(tx.TraceId, tx)
		if err != nil {
			return err
		}
	}
	return nil
}
