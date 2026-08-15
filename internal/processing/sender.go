package processing

import (
	"context"
	"time"

	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	"github.com/Nox1KCL/Arbitrage/internal/transport"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type SendingServer struct {
	Client pb.DataServiceClient
	Obs    *telemetry.Observe
}

func (s *SendingServer) Sending(ctx context.Context, payload chan []*transport.FormedMessage) {
	metrics, err := NewProcessingMetrics(s.Obs.Meter)
	if err != nil {
		plog.ErrorContext(ctx, "could not init processing metrics", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case msgs := <-payload:
			childCtx, span := s.Obs.Tracer.Start(ctx, "SendingProcess")
			s.sendingProcess(childCtx, msgs, metrics)
			span.End()
		}
	}
}

func (s *SendingServer) sendingProcess(ctx context.Context, msgs []*transport.FormedMessage, m *processingMetrics) {
	for _, msg := range msgs {
		req := &pb.AlertNotification{
			TelegramChatId: msg.TelegramUserID,
			Text:           msg.Text,
		}

		res := s.gRPCsender(ctx, req, m)
		if res != nil && !res.GetStatus() {
			plog.Warn("Delivery rejected message", "details", res.GetDetails())
			m.counter.Add(ctx, 1, metric.WithAttributes(
				attribute.String("stage", "msg.delivery"),
				attribute.String("type", "grpc.sending.error"),
			))
		}
	}
}

func (s *SendingServer) gRPCsender(ctx context.Context, req *pb.AlertNotification, m *processingMetrics) *pb.Ack {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		m.histogram.Record(ctx, duration, metric.WithAttributes(
			attribute.String("stage", "sending"),
		))
	}()

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
