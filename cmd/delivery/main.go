package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Nox1KCL/Arbitrage/internal/config"
	"github.com/Nox1KCL/Arbitrage/internal/delivery"
	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	"github.com/Nox1KCL/Arbitrage/internal/syncutils"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var dlog = slog.With("service", "delivery")

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

	metrics, err := delivery.NewDeliveryMetrics(observer.Meter)
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
	pb.RegisterDataServiceServer(grpcServer, &delivery.Server{
		Obs:     observer,
		Metrics: metrics,
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
