package mtg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fox-one/mixin-sdk-go"
)

const (
	CollectibleMetaTokenId  = "2f8aa18a-3cb8-31d5-95bc-5a4f2e25dc2f"
	CollectibleMixinAssetId = "1700941284a95f31b25ec8c546008f208f88eee4419ccdcdbe6e3195e60128ca"
)

type CollectibleOutput struct {
	Type               string    `json:"type"`
	UserId             string    `json:"user_id"`
	OutputId           string    `json:"output_id"`
	TokenId            string    `json:"token_id"`
	TransactionHash    string    `json:"transaction_hash"`
	OutputIndex        int64     `json:"output_index"`
	Amount             string    `json:"amount"`
	SendersThreshold   int64     `json:"senders_threshold"`
	Senders            []string  `json:"senders"`
	ReceiversThreshold int64     `json:"receivers_threshold"`
	Receivers          []string  `json:"receivers"`
	Memo               string    `json:"memo"`
	StateName          string    `json:"state"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	SignedBy           string    `json:"signed_by"`
	SignedTx           string    `json:"signed_tx"`

	State int `json:"-"`
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
		switch o.StateName {
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

func (grp *Group) signCollectibleTransaction(ctx context.Context, tx *Transaction) ([]byte, error) {
	panic(0)
}
