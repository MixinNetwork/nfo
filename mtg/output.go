package mtg

import (
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/shopspring/decimal"
)

const (
	OutputStateUnspent = 10
	OutputStateSigned  = 11
	OutputStateSpent   = 12
)

type Output struct {
	GroupId         string
	UserID          string
	UTXOID          string
	AssetID         string
	TransactionHash crypto.Hash
	OutputIndex     int
	Sender          string
	Amount          decimal.Decimal
	Threshold       uint8
	Members         []string
	Memo            string
	State           int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	SignedBy        string
	SignedTx        string
}

func NewOutputFromMultisig(utxo *mixin.MultisigUTXO) *Output {
	out := &Output{
		UserID:      utxo.UserID,
		UTXOID:      utxo.UTXOID,
		AssetID:     utxo.AssetID,
		OutputIndex: utxo.OutputIndex,
		Sender:      utxo.Sender,
		Amount:      utxo.Amount,
		Threshold:   utxo.Threshold,
		Members:     utxo.Members,
		Memo:        utxo.Memo,
		CreatedAt:   utxo.CreatedAt,
		UpdatedAt:   utxo.UpdatedAt,
		SignedBy:    utxo.SignedBy,
		SignedTx:    utxo.SignedTx,
	}
	out.TransactionHash = crypto.Hash(utxo.TransactionHash)
	switch utxo.State {
	case mixin.UTXOStateUnspent:
		out.State = OutputStateUnspent
	case mixin.UTXOStateSigned:
		out.State = OutputStateSigned
	case mixin.UTXOStateSpent:
		out.State = OutputStateSpent
	}
	return out
}

func (out *Output) StateName() string {
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
