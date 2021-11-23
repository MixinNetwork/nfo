package mtg

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go"
)

type Group struct {
	mixin   *mixin.Client
	store   Store
	workers []Worker

	id        string
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
		id:    generateGenesisId(conf),
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

func (grp *Group) GenesisId() string {
	return grp.id
}

func (grp *Group) GetMembers() []string {
	return grp.members
}

func (grp *Group) GetThreshold() int {
	return grp.threshold
}

func (grp *Group) AddWorker(wkr Worker) {
	grp.workers = append(grp.workers, wkr)
}

func (grp *Group) Run(ctx context.Context) {
	logger.Printf("Group(%s, %d).Run(v0.0.4)\n", mixin.HashMembers(grp.members), grp.threshold)
	go grp.loopCollectibles(ctx)
	grp.loopMultsigis(ctx)
}

func (grp *Group) loopMultsigis(ctx context.Context) {
	for {
		// drain all the utxos in the order of updated time
		grp.drainOutputsFromNetwork(ctx, 100)

		// handle the utxos queue by created time
		grp.handleActionsQueue(ctx)

		// sing any possible transactions from BuildTransaction
		grp.signTransactions(ctx)

		// publish all signed transactions to the mainnet
		grp.publishTransactions(ctx)
	}
}

func (grp *Group) loopCollectibles(ctx context.Context) {
	for {
		grp.drainCollectibleOutputsFromNetwork(ctx, 100)
		grp.handleCollectibleActionsQueue(ctx)
		grp.signCollectibleTransactions(ctx)
		grp.publishCollectibleTransactions(ctx)
	}
}

// FIXME sign one transaction per loop, slow
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
	ver, _ := common.UnmarshalVersionedTransaction(raw)
	tx.Raw = raw
	tx.Hash = ver.PayloadHash()
	tx.UpdatedAt = time.Now()
	tx.State = TransactionStateSigning

	extra, _ := base64.RawURLEncoding.DecodeString(string(ver.Extra))
	var p mixinExtraPack
	err = common.MsgpackUnmarshal(extra, &p)
	if p.T.String() != tx.TraceId {
		panic(hex.EncodeToString(raw))
	}

	return grp.store.WriteTransaction(tx)
}

func (grp *Group) publishTransactions(ctx context.Context) error {
	txs, err := grp.store.ListTransactions(TransactionStateSigned, 0)
	if err != nil || len(txs) == 0 {
		return err
	}
	for _, tx := range txs {
		snapshot, err := grp.snapshotTransaction(ctx, tx.Raw)
		if err != nil {
			return err
		} else if !snapshot {
			continue
		}
		tx.State = TransactionStateSnapshot
		err = grp.store.WriteTransaction(tx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (grp *Group) compactOutputs(ctx context.Context) {
	panic("TODO")
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
	ver, _ := common.UnmarshalVersionedTransaction(raw)
	tx.Raw = raw
	tx.Hash = ver.PayloadHash()
	tx.UpdatedAt = time.Now()
	tx.State = TransactionStateSigning

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
		snapshot, err := grp.snapshotTransaction(ctx, tx.Raw)
		if err != nil {
			return err
		} else if !snapshot {
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

func (grp *Group) snapshotTransaction(ctx context.Context, b []byte) (bool, error) {
	raw := hex.EncodeToString(b)
	h, err := grp.mixin.SendRawTransaction(ctx, raw)
	logger.Verbosef("Group.snapshotTransaction(%s) => %s, %v", raw, h, err)
	if err != nil {
		return false, err
	}
	s, err := grp.mixin.GetRawTransaction(ctx, *h)
	if err != nil {
		return false, err
	}
	return s.Snapshot != nil && s.Snapshot.HasValue(), nil
}

func generateGenesisId(conf *Configuration) string {
	sort.Slice(conf.Genesis.Members, func(i, j int) bool {
		return conf.Genesis.Members[i] < conf.Genesis.Members[j]
	})
	id := strings.Join(conf.Genesis.Members, "")
	id = fmt.Sprintf("%s:%d:%d", id, conf.Genesis.Threshold, conf.Genesis.Timestamp)
	return crypto.NewHash([]byte(id)).String()
}
