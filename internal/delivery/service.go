package delivery

import (
	"context"
	"fmt"
	"log/slog"

	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var dlog = slog.With("service", "delivery")

type deliveryMetrics struct {
	counter   metric.Int64Counter
	histogram metric.Float64Histogram
}

func NewDeliveryMetrics(meter metric.Meter) (*deliveryMetrics, error) {
	counter, err := meter.Int64Counter(
		"counter",
		metric.WithDescription("for counting sent messages, errors"),
	)
	if err != nil {
		err := fmt.Errorf("counter init failed: %w", err)
		return nil, err
	}

	histogram, err := meter.Float64Histogram(
		"histogram",
		metric.WithDescription("for calculating averange stats"),
	)
	if err != nil {
		err := fmt.Errorf("histogram init failed: %w", err)
		return nil, err
	}

	metrics := &deliveryMetrics{
		counter:   counter,
		histogram: histogram,
	}
	return metrics, nil
}

type Server struct {
	pb.UnimplementedDataServiceServer
	Obs     *telemetry.Observe
	Metrics *deliveryMetrics
}

func (s *Server) SendUser(ctx context.Context, req *pb.AlertNotification) (*pb.Ack, error) {
	if err := sendTelegramMessage(ctx, req.GetTelegramChatId(), req.GetText(), s.Obs, s.Metrics); err != nil {
		dlog.Error("Could not send telegram message",
			"error", err,
			"chat_id", req.GetTelegramChatId(),
		)
		s.Metrics.counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("stage", "sendTelegramMessage"),
			attribute.String("type", "grpc.sent.error"),
		))

		return &pb.Ack{
			Status:  false,
			Details: err.Error(),
		}, nil
	}

	s.Metrics.counter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("stage", "sendTelegramMessage"),
		attribute.String("type", "grpc.sent.success"),
	))

	return &pb.Ack{
		Status:  true,
		Details: "",
	}, nil
}
