package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/nfo/nft"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

const (
	CNBAssetID = "965e5c6e-434c-3fa9-b780-c50f43cd955c"
)

// Messenger is a simple MTG worker demo, it also sends some cmd to MM
type MessengerWorker struct {
	client *mixin.Client
	grp    *mtg.Group
}

func NewMessengerWorker(ctx context.Context, grp *mtg.Group, conf *mtg.Configuration) *MessengerWorker {
	s := &mixin.Keystore{
		ClientID:   conf.App.ClientId,
		SessionID:  conf.App.SessionId,
		PrivateKey: conf.App.PrivateKey,
		PinToken:   conf.App.PinToken,
	}
	client, err := mixin.NewFromKeystore(s)
	if err != nil {
		panic(err)
	}
	rand.Seed(time.Now().UnixNano())
	rw := &MessengerWorker{
		client: client,
		grp:    grp,
	}
	go rw.loop(ctx)
	return rw
}

func (rw *MessengerWorker) ProcessOutput(ctx context.Context, out *mtg.Output) {
	if out.Sender == "" || out.AssetID != CNBAssetID {
		return
	}
	receivers := []string{out.Sender}
	traceId := mixin.UniqueConversationID(out.UTXOID, "refund")
	err := rw.grp.BuildTransaction(ctx, out.AssetID, receivers, 1, out.Amount.String(), "refund", traceId)
	if err != nil {
		panic(err)
	}
}

func (rw *MessengerWorker) loop(ctx context.Context) {
	for {
		err := rw.client.LoopBlaze(context.Background(), rw)
		fmt.Println("LoopBlaze", err)
		if ctx.Err() != nil {
			break
		}
		time.Sleep(3 * time.Second)
	}
}

func (rw *MessengerWorker) OnMessage(ctx context.Context, msg *mixin.MessageView, userId string) error {
	code := "mixin://codes/"
	if msg.Category == mixin.MessageCategoryPlainSticker {
		pid, err := rw.handleMintMessage(ctx, msg.MessageID)
		if err != nil {
			return nil
		}
		code = code + pid
	} else if msg.Category == mixin.MessageCategoryPlainText {
		pid, err := rw.handleRefundMessage(ctx, msg.MessageID)
		if err != nil {
			return nil
		}
		code = code + pid
	} else {
		return nil
	}
	mr := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		Category:       mixin.MessageCategoryPlainText,
		MessageID:      mixin.UniqueConversationID(code, code),
		Data:           base64.RawURLEncoding.EncodeToString([]byte(code)),
	}
	return rw.client.SendMessage(ctx, mr)
}

func (rw *MessengerWorker) OnAckReceipt(ctx context.Context, msg *mixin.MessageView, userId string) error {
	return nil
}

func (rw *MessengerWorker) handleMintMessage(ctx context.Context, msgId string) (string, error) {
	amount, err := decimal.NewFromString(nft.MintMinimumCost)
	if err != nil {
		return "", err
	}
	mid, err := uuid.FromString(msgId)
	if err != nil {
		return "", err
	}
	contentHash := crypto.NewHash([]byte("TEST:" + msgId))
	nfo := nft.BuildMintNFO(nft.NMDefaultGroupKey, mid.Bytes(), contentHash)
	pr := mixin.TransferInput{
		AssetID: nft.MintAssetId,
		Amount:  amount,
		TraceID: msgId,
		Memo:    base64.RawURLEncoding.EncodeToString(nfo),
	}
	pr.OpponentMultisig.Receivers = rw.grp.GetMembers()
	pr.OpponentMultisig.Threshold = uint8(rw.grp.GetThreshold())
	payment, err := rw.client.VerifyPayment(ctx, pr)
	if err != nil {
		return "", err
	}
	return payment.CodeID, nil
}

func (rw *MessengerWorker) handleRefundMessage(ctx context.Context, msgId string) (string, error) {
	amount, err := decimal.NewFromString(fmt.Sprint(rand.Intn(10000)))
	if err != nil {
		return "", err
	}
	pr := mixin.TransferInput{
		AssetID: CNBAssetID,
		Amount:  amount,
		TraceID: msgId,
		Memo:    "REFUND",
	}
	pr.OpponentMultisig.Receivers = rw.grp.GetMembers()
	pr.OpponentMultisig.Threshold = uint8(rw.grp.GetThreshold())
	payment, err := rw.client.VerifyPayment(ctx, pr)
	if err != nil {
		return "", err
	}
	return payment.CodeID, nil
}