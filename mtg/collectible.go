package mtg

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/shopspring/decimal"
)

const (
	CollectibleMetaTokenId  = "2f8aa18a-3cb8-31d5-95bc-5a4f2e25dc2f"
	CollectibleMixinAssetId = "1700941284a95f31b25ec8c546008f208f88eee4419ccdcdbe6e3195e60128ca"
)

type CollectibleOutput struct {
	Type               string      `json:"type"`
	UserId             string      `json:"user_id"`
	OutputId           string      `json:"output_id"`
	TokenId            string      `json:"token_id"`
	TransactionHash    crypto.Hash `json:"transaction_hash"`
	OutputIndex        int         `json:"output_index"`
	Amount             string      `json:"amount"`
	SendersThreshold   int64       `json:"senders_threshold"`
	Senders            []string    `json:"senders"`
	ReceiversThreshold int64       `json:"receivers_threshold"`
	Receivers          []string    `json:"receivers"`
	Memo               string      `json:"memo"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
	SignedBy           string      `json:"signed_by"`
	SignedTx           string      `json:"signed_tx"`

	JsonState string `json:"state" msgpack:"-"`
	State     int    `json:"-"`
}

type CollectibleTransaction struct {
	TraceId   string
	State     int
	Receivers []string
	Threshold int
	Amount    string
	NFO       []byte
	Raw       []byte
	UpdatedAt time.Time
}

func (grp *Group) BuildCollectibleMintTransaction(ctx context.Context, receiver string, nfo []byte) error {
	traceId := nfoTraceId(nfo)
	old, err := grp.store.ReadCollectibleTransaction(traceId)
	if err != nil || old != nil {
		return err
	}
	tx := &CollectibleTransaction{
		TraceId:   traceId,
		State:     TransactionStateInitial,
		Receivers: []string{receiver},
		Threshold: 1,
		Amount:    "1",
		NFO:       nfo,
		UpdatedAt: time.Now(),
	}
	return grp.store.WriteCollectibleTransaction(tx.TraceId, tx)
}

func (grp *Group) ReadCollectibleOutputs(ctx context.Context, members []string, threshold uint8, offset time.Time, limit int) ([]*CollectibleOutput, error) {
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

	var outputs []*CollectibleOutput
	err := grp.mixin.Get(ctx, "/collectibles/outputs", params, &outputs)
	if err != nil {
		return nil, err
	}

	for _, o := range outputs {
		switch o.JsonState {
		case mixin.UTXOStateUnspent:
			o.State = OutputStateUnspent
		case mixin.UTXOStateSigned:
			o.State = OutputStateSigned
		case mixin.UTXOStateSpent:
			o.State = OutputStateSpent
		}
	}
	return outputs, nil
}

func (out *CollectibleOutput) StateName() string {
	switch out.State {
	case OutputStateUnspent:
		return mixin.UTXOStateUnspent
	case OutputStateSigned:
		return mixin.UTXOStateSigned
	case OutputStateSpent:
		return mixin.UTXOStateSpent
	}
	panic(out.State)
}

func (grp *Group) signCollectibleMintTransaction(ctx context.Context, tx *CollectibleTransaction) ([]byte, error) {
	outputs, err := grp.store.ListCollectibleOutputsForTransaction(tx.TraceId)
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		outputs, err = grp.store.ListCollectibleOutputsForToken(mixin.UTXOStateUnspent, CollectibleMetaTokenId, 1)
	}
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		return nil, fmt.Errorf("empty outputs %s", tx.Amount)
	}

	ver := decodeCollectibleTransaction(outputs[0].SignedTx)
	if ver == nil {
		ver, err = grp.buildRawCollectibleMintTransaction(ctx, tx, outputs)
		if err != nil {
			return nil, err
		}
	} else if ver.AggregatedSignature != nil {
		return ver.Marshal(), nil
	}

	raw := hex.EncodeToString(ver.AsLatestVersion().Marshal())
	req, err := grp.CreateCollectibleRequest(ctx, mixin.MultisigActionSign, raw)
	if err != nil {
		return nil, err
	}

	for _, out := range outputs {
		out.State = OutputStateSigned
		out.SignedBy = ver.AsLatestVersion().PayloadHash().String()
		out.SignedTx = raw
	}
	err = grp.store.WriteCollectibleOutputs(outputs, tx.TraceId)
	if err != nil {
		return nil, err
	}

	req, err = grp.SignCollectible(ctx, req.RequestID, grp.pin)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(req.RawTransaction)
}

func (grp *Group) buildRawCollectibleMintTransaction(ctx context.Context, tx *CollectibleTransaction, outputs []*CollectibleOutput) (*common.VersionedTransaction, error) {
	if tx.Amount != "1" {
		panic(tx.Amount)
	}
	assetId, err := crypto.HashFromString(CollectibleMixinAssetId)
	if err != nil {
		panic(err)
	}
	ver := common.NewTransaction(assetId)
	ver.Extra = tx.NFO

	var total common.Integer
	for _, out := range outputs {
		total = total.Add(common.NewIntegerFromString(out.Amount))
		ver.AddInput(out.TransactionHash, out.OutputIndex)
	}
	if total.Cmp(common.NewIntegerFromString(tx.Amount)) < 0 {
		return nil, fmt.Errorf("insufficient %s %s", total, tx.Amount)
	}

	keys, err := grp.mixin.BatchReadGhostKeys(ctx, []*mixin.GhostInput{{
		Receivers: tx.Receivers,
		Index:     0,
		Hint:      tx.TraceId,
	}, {
		Receivers: grp.members,
		Index:     1,
		Hint:      tx.TraceId,
	}})
	if err != nil {
		return nil, err
	}

	amount, err := decimal.NewFromString(tx.Amount)
	if err != nil {
		return nil, err
	}
	out := keys[0].DumpOutput(uint8(tx.Threshold), amount)
	ver.Outputs = append(ver.Outputs, newCommonOutput(out))

	if diff := total.Sub(common.NewIntegerFromString(tx.Amount)); diff.Sign() > 0 {
		amount, err := decimal.NewFromString(diff.String())
		if err != nil {
			return nil, err
		}
		out := keys[1].DumpOutput(uint8(grp.threshold), amount)
		ver.Outputs = append(ver.Outputs, newCommonOutput(out))
	}

	return ver.AsLatestVersion(), nil
}

func decodeCollectibleTransaction(s string) *common.VersionedTransaction {
	raw, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	tx, err := common.UnmarshalVersionedTransaction(raw)
	if err != nil {
		return nil
	}
	return tx
}

func nfoTraceId(nfo []byte) string {
	nid := crypto.NewHash(nfo).String()
	return mixin.UniqueConversationID(nid, nid)
}

type cr struct {
	RequestID      string `json:"request_id"`
	RawTransaction string `json:"raw_transaction"`
}

func (grp *Group) CreateCollectibleRequest(ctx context.Context, action, raw string) (*cr, error) {
	params := map[string]string{
		"action": action,
		"raw":    raw,
	}

	var req cr
	err := grp.mixin.Post(ctx, "/collectibles/requests", params, &req)
	return &req, err
}

func (grp *Group) SignCollectible(ctx context.Context, reqID, pin string) (*cr, error) {
	uri := "/collectibles/requests/" + reqID + "/sign"
	params := map[string]string{
		"pin": grp.mixin.EncryptPin(pin),
	}

	var req cr
	err := grp.mixin.Post(ctx, uri, params, &req)
	return &req, err
}
