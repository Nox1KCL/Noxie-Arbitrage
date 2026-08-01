package processing

import (
	"context"
	"log/slog"
	"time"

	"github.com/Nox1KCL/Arbitrage/internal/transport"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
)

var selog = slog.With("service", "processing")

func Sending(ctx context.Context, client pb.DataServiceClient, payload chan []*transport.FormedMessage) {
	for {
		select {
		case <-ctx.Done():
			return
		case msgs := <-payload:
			for _, m := range msgs {
				req := &pb.AlertNotification{
					TelegramChatId: m.TelegramUserID,
					Text:           m.Text,
				}
				res := gRPCsender(ctx, client, req)
				if res != nil && !res.GetStatus() {
					selog.Warn("Delivery rejected message", "details", res.GetDetails())
				}
			}
		}
	}
}

func gRPCsender(ctx context.Context, client pb.DataServiceClient, req *pb.AlertNotification) *pb.Ack {
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := client.SendUser(callCtx, req)
	if err != nil {
		selog.Error("Could not send user", "error", err)
		return nil
	}
	return res
}
