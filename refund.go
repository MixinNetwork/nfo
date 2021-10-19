package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"time"

	"github.com/MixinNetwork/nfo/mtg"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/shopspring/decimal"
)

const (
	CNBAssetID = "965e5c6e-434c-3fa9-b780-c50f43cd955c"
)

type RefundWorker struct {
	client *mixin.Client
	grp    *mtg.Group
}

func NewRefundWorker(ctx context.Context, grp *mtg.Group, conf *mtg.Configuration) *RefundWorker {
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
	rw := &RefundWorker{
		client: client,
		grp:    grp,
	}
	go rw.loop(ctx)
	return rw
}

func (rw *RefundWorker) ProcessOutput(ctx context.Context, out *mtg.Output) {
	receivers := []string{out.Sender}
	traceId := mixin.UniqueConversationID(out.UTXOID, "refund")
	err := rw.grp.BuildTransaction(ctx, out.AssetID, receivers, 1, out.Amount.String(), "refund", traceId)
	if err != nil {
		panic(err)
	}
}

func (rw *RefundWorker) loop(ctx context.Context) {
	for {
		err := rw.client.LoopBlaze(context.Background(), rw)
		fmt.Println("LoopBlaze", err)
		if ctx.Err() != nil {
			break
		}
		time.Sleep(3 * time.Second)
	}
}

func (rw *RefundWorker) OnMessage(ctx context.Context, msg *mixin.MessageView, userId string) error {
	if msg.Category != mixin.MessageCategoryPlainText {
		return nil
	}
	pid, err := rw.handleMessage(ctx, msg.MessageID)
	if err != nil {
		return nil
	}
	code := "mixin://codes/" + pid
	mr := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		Category:       mixin.MessageCategoryPlainText,
		MessageID:      mixin.UniqueConversationID(pid, pid),
		Data:           base64.RawURLEncoding.EncodeToString([]byte(code)),
	}
	return rw.client.SendMessage(ctx, mr)
}

func (rw *RefundWorker) OnAckReceipt(ctx context.Context, msg *mixin.MessageView, userId string) error {
	return nil
}

func (rw *RefundWorker) handleMessage(ctx context.Context, msgId string) (string, error) {
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