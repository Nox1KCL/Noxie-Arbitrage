package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"

	"github.com/Nox1KCL/Arbitrage/internal/broker"
	"github.com/Nox1KCL/Arbitrage/internal/config"
	"github.com/Nox1KCL/Arbitrage/internal/database"
	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	"github.com/Nox1KCL/Arbitrage/internal/processing"
	"github.com/Nox1KCL/Arbitrage/internal/syncutils"
	"github.com/Nox1KCL/Arbitrage/internal/transport"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
)

var plog = slog.With("service", "processing")

type server struct {
	pb.UnimplementedProcessingServiceServer
	db    *gorm.DB
	store *processing.SubscriptionStore
}

func (s *server) ReloadSubscriptions(ctx context.Context, _ *emptypb.Empty) (*pb.ReloadSubscriptionsResponse, error) {
	err := s.store.Reload(s.db)
	if err != nil {
		return &pb.ReloadSubscriptionsResponse{Status: false}, err
	}
	return &pb.ReloadSubscriptionsResponse{
		Status: true,
	}, nil
}

func main() {
	syncutils.LoadEnv()

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	db, err := database.Connect()
	if err != nil {
		plog.ErrorContext(ctx, "Could not connect to db", "error", err)
		os.Exit(1)
	}
	if err := db.AutoMigrate(&models.Subscription{}); err != nil {
		plog.ErrorContext(ctx, "Could not create a table", "error", err)
		os.Exit(1)
	}
	store := &processing.SubscriptionStore{}
	err = store.Reload(db)
	if err != nil {
		plog.ErrorContext(ctx, "Could not reload store", "error", err)
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", ":50050")
	if err != nil {
		plog.ErrorContext(ctx, "Could not listen on port 50050", "error", err)
		os.Exit(1)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterProcessingServiceServer(grpcServer, &server{db: db, store: store})
	reflection.Register(grpcServer)

	plog.Info("grpc server started successfully")

	var serviceWg syncutils.MyWaitGroup
	serviceWg.Go(func() {
		if err := grpcServer.Serve(listener); err != nil {
			plog.ErrorContext(ctx, "grpc server error", "error", err)
			os.Exit(1)
		}
	})

	conn, err := grpc.NewClient("delivery:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		plog.ErrorContext(ctx, "Could not connect to client", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewDataServiceClient(conn)

	b, err := broker.NewBroker("parser.binance.ticker")
	if err != nil {
		plog.ErrorContext(ctx, "Could not create broker", "error", err)
		os.Exit(1)
	}

	cfg, err := config.GetConfig("config.toml")
	if err != nil {
		plog.ErrorContext(ctx, "getting config", "error", err)
		os.Exit(1)
	}

	shutdown, obs, err := telemetry.NewTelemetry(&cfg.LumberConfig)
	if err != nil {
		plog.ErrorContext(ctx, "getting telemetry", "error", err)
		os.Exit(1)
	}
	defer shutdown(ctx)

	msgs, err := b.Unload()
	if err != nil {
		plog.ErrorContext(ctx, "Could not unload broker", "error", err)
		os.Exit(1)
	}

	tickerChannel := make(chan processing.TickerForm)
	serviceWg.Go(func() {
		plog.InfoContext(ctx, "Starting scanner process")
		processing.Scanner(ctx, msgs, tickerChannel, obs)
	})

	payload := make(chan []*transport.FormedMessage)
	serviceWg.Go(func() {
		plog.InfoContext(ctx, "Starting filter process")
		processing.Filter(ctx, tickerChannel, payload, store, obs)
	})

	s := &processing.SendingServer{
		Client: client,
		Obs:    obs,
	}

	serviceWg.Go(func() {
		plog.InfoContext(ctx, "Starting sending process")
		s.Sending(ctx, payload)
	})

	sig := <-sigChan
	plog.WarnContext(ctx, "Received signal", "signal", sig)

	cancel()
	serviceWg.Wait()
}
