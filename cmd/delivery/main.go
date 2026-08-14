package main

import (
	"context"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var dlog = slog.With("service", "delivery")

type server struct {
	pb.UnimplementedDataServiceServer
}

// TODO: Придумати як застовувати сервіс тут, також закомітити зміни
func (s *server) SendUser(ctx context.Context, req *pb.AlertNotification) (*pb.Ack, error) {
	if err := sendTelegramMessage(ctx, req.GetTelegramChatId(), req.GetText()); err != nil {
		dlog.Error("Could not send telegram message",
			"error", err,
			"chat_id", req.GetTelegramChatId(),
		)

		return &pb.Ack{
			Status:  false,
			Details: err.Error(),
		}, nil
	}

	return &pb.Ack{
		Status:  true,
		Details: "",
	}, nil
}

func main() {
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

	_, span := observer.Tracer.Start(ctx, "DeliveryInit")
	defer span.End()

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		dlog.Error("Could not listen on port 50051", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDataServiceServer(grpcServer, &server{})
	reflection.Register(grpcServer)

	dlog.Info("grpc server started successfully")

	var wg syncutils.MyWaitGroup
	wg.Go(func() {
		span.AddEvent("Starting grpc server")
		if err := grpcServer.Serve(listener); err != nil {
			dlog.Error("grpc server error", "error", err)
		}
	})

	sig := <-sigChan
	dlog.WarnContext(ctx, "Received signal", "signal", sig)
	cancel()
	wg.Wait()
	span.AddEvent("recieved signal, gracefully shutdown..")
	span.SetAttributes(
		attribute.String("os.signal.name", sig.String()),
	)
	span.End()
}
