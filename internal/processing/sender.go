package processing

import (
	"context"
	"log/slog"
	"time"

	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	"github.com/Nox1KCL/Arbitrage/internal/transport"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
)

var selog = slog.With("service", "processing")

type SendingServer struct {
	Client pb.DataServiceClient
	Obs    *telemetry.Observe
}

func Sending(ctx context.Context, payload chan []*transport.FormedMessage, s *SendingServer) {
	for {
		select {
		case <-ctx.Done():
			return
		case msgs := <-payload:
			childCtx, span := s.Obs.Tracer.Start(ctx, "SendingProcess")
			sendingProcess(childCtx, msgs, s)
			span.End()
		}
	}
}

func sendingProcess(ctx context.Context, msgs []*transport.FormedMessage, s *SendingServer) {
	for _, m := range msgs {
		req := &pb.AlertNotification{
			TelegramChatId: m.TelegramUserID,
			Text:           m.Text,
		}

		res := gRPCsender(ctx, req, s)
		if res != nil && !res.GetStatus() {
			selog.Warn("Delivery rejected message", "details", res.GetDetails())
		}
	}
}

func gRPCsender(ctx context.Context, req *pb.AlertNotification, s *SendingServer) *pb.Ack {
	childCtx, span := s.Obs.Tracer.Start(ctx, "grpcRequest")
	defer span.End()

	callCtx, cancel := context.WithTimeout(childCtx, 5*time.Second)
	defer cancel()

	res, err := s.Client.SendUser(callCtx, req)
	if err != nil {
		span.AddEvent("failed delivery message to user")
		span.RecordError(err)

		return nil
	}
	return res
}
