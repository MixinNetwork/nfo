package mtg

import (
	"context"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/shopspring/decimal"
)

type CollectibleOutput struct {
	UserID             string
	OutputID           string
	TokenID            string
	TransactionHash    crypto.Hash
	OutputIndex        int
	Amount             decimal.Decimal
	Senders            []string
	SendersThreshold   uint8
	Receivers          []string
	ReceiversThreshold uint8
	Memo               string
	State              int
	CreatedAt          time.Time
	UpdatedAt          time.Time
	SignedBy           string
	SignedTx           string
}

func (grp *Group) ReadCollectibleOutputs(ctx context.Context, members []string, threshold uint8, offset time.Time, batch int) ([]*CollectibleOutput, error) {
	return nil, nil
}

func (grp *Group) signCollectibleTransaction(ctx context.Context, tx *Transaction) ([]byte, error) {
	panic(0)
}
