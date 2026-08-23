package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Nox1KCL/Arbitrage/internal/config"
	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	"github.com/Nox1KCL/Arbitrage/internal/syncutils"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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

type server struct {
	pb.UnimplementedDataServiceServer
	obs     *telemetry.Observe
	metrics *deliveryMetrics
}

// TODO: Придумати як застовувати сервіс тут, також закомітити зміни
func (s *server) SendUser(ctx context.Context, req *pb.AlertNotification) (*pb.Ack, error) {
	if err := sendTelegramMessage(ctx, req.GetTelegramChatId(), req.GetText(), s.obs, s.metrics); err != nil {
		dlog.Error("Could not send telegram message",
			"error", err,
			"chat_id", req.GetTelegramChatId(),
		)
		s.metrics.counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("stage", "sendTelegramMessage"),
			attribute.String("type", "grpc.sent.error"),
		))

		return &pb.Ack{
			Status:  false,
			Details: err.Error(),
		}, nil
	}

	s.metrics.counter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("stage", "sendTelegramMessage"),
		attribute.String("type", "grpc.sent.success"),
	))

	return &pb.Ack{
		Status:  true,
		Details: "",
	}, nil
}

func main() {
	syncutils.LoadEnv()

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	cfg, err := config.GetConfig("config.toml")
	if err != nil {
		dlog.ErrorContext(ctx, "Could not load config", "error", err)
		os.Exit(1)
	}

	shutdown, observer, err := telemetry.NewTelemetry(&cfg.LumberConfig)
	if err != nil {
		dlog.ErrorContext(ctx, "Could not get observer", "error", err)
		os.Exit(1)
	}
	defer shutdown(ctx)

	metrics, err := NewDeliveryMetrics(observer.Meter)
	if err != nil {
		dlog.ErrorContext(ctx, "getting metrics", "error", err)
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		dlog.ErrorContext(ctx, "Could not listen on port 50051", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDataServiceServer(grpcServer, &server{
		obs:     observer,
		metrics: metrics,
	})
	reflection.Register(grpcServer)

	dlog.InfoContext(ctx, "grpc server started successfully")

	var wg syncutils.MyWaitGroup
	wg.Go(func() {
		dlog.InfoContext(ctx, "starting grpc server")
		if err := grpcServer.Serve(listener); err != nil {
			dlog.ErrorContext(ctx, "grpc server error", "error", err)
		}
	})

	sig := <-sigChan
	dlog.WarnContext(ctx, "Received signal", "signal", sig)
	cancel()
	wg.Wait()
}
