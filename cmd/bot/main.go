package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nox1KCL/Arbitrage/internal/config"
	"github.com/Nox1KCL/Arbitrage/internal/database"
	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	"github.com/Nox1KCL/Arbitrage/internal/syncutils"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

var blog = slog.With("service", "bot")

type botMetrics struct {
	sentMessages        metric.Int64Counter
	dbErrors            metric.Int64Counter
	activeSubscriptions metric.Int64UpDownCounter
	requestsErrors      metric.Int64Counter
	clientErrors        metric.Int64Counter
}

func newBotMetrics(meter metric.Meter) (*botMetrics, error) {
	sentMessages, err := meter.Int64Counter(
		"sent_messages",
		metric.WithDescription("Total count of sent messages"),
	)
	if err != nil {
		err := fmt.Errorf("sentMessage metric init failed: %w", err)
		return nil, err
	}

	dbErrors, err := meter.Int64Counter(
		"db_errors_total",
		metric.WithDescription("Total count of db errors"),
	)
	if err != nil {
		err := fmt.Errorf("dbErrors metric init failed: %w", err)
		return nil, err
	}

	activeSubscriptions, err := meter.Int64UpDownCounter(
		"active_subscriptions",
		metric.WithDescription("Total count of active subscription"),
	)
	if err != nil {
		err := fmt.Errorf("activeSubscriptions metric init failed: %w", err)
		return nil, err
	}

	requestsErrors, err := meter.Int64Counter(
		"requests_errors_count",
		metric.WithDescription("Total count of telegram requests count"),
	)
	if err != nil {
		err := fmt.Errorf("requestsErrors metric init failed: %w", err)
		return nil, err
	}

	clientErrors, err := meter.Int64Counter(
		"client_errors_count",
		metric.WithDescription("Total count of client errors"),
	)
	if err != nil {
		err := fmt.Errorf("clientErrors metric init failed: %w", err)
		return nil, err
	}

	return &botMetrics{
		sentMessages:        sentMessages,
		dbErrors:            dbErrors,
		activeSubscriptions: activeSubscriptions,
		requestsErrors:      requestsErrors,
		clientErrors:        clientErrors,
	}, nil
}

type botService struct {
	observer *telemetry.Observe
	metrics  *botMetrics
	db       *gorm.DB
	token    string
	client   pb.ProcessingServiceClient
}

func NewBotService(obs *telemetry.Observe, db *gorm.DB, token string, client pb.ProcessingServiceClient) (*botService, error) {
	m, err := newBotMetrics(obs.Meter)
	if err != nil {
		return nil, fmt.Errorf("failed to init bot botMetrics: %w", err)
	}

	return &botService{
		observer: obs,
		metrics:  m,
		db:       db,
		token:    token,
		client:   client,
	}, nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	token := os.Getenv("TELEGRAM_BOT_API")
	if token == "" {
		blog.ErrorContext(ctx, "TELEGRAM_BOT_API is not set")
		os.Exit(1)
	}

	conn, err := grpc.NewClient("processing-go:50050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		blog.ErrorContext(ctx, "Could not connect to client", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewProcessingServiceClient(conn)
	blog.InfoContext(ctx, "grpc server connection finished successfully")

	db, err := database.Connect()
	if err != nil {
		blog.ErrorContext(ctx, "Could not connect to db", "error", err)
		os.Exit(1)
	}
	if err := db.AutoMigrate(&models.Subscription{}); err != nil {
		blog.ErrorContext(ctx, "Could not create a table", "error", err)
		os.Exit(1)
	}
	blog.Info("db init finished successfully")

	cfg, err := config.GetConfig("config.toml")
	if err != nil {
		blog.ErrorContext(ctx, "Could not load config", "error", err)
		os.Exit(1)
	}

	shutdown, observer, err := telemetry.NewTelemetry(&cfg.LumberConfig)
	if err != nil {
		blog.ErrorContext(ctx, "Could not get observer", "error", err)
		os.Exit(1)
	}
	defer shutdown(ctx)
	blog.InfoContext(ctx, "telemetry init finished successfully")

	service, err := NewBotService(observer, db, token, client)
	if err != nil {
		blog.ErrorContext(ctx, "Could not get bot service", "error", err)
		os.Exit(1)
	}
	_, span := service.observer.Tracer.Start(ctx, "BotInit")
	blog.InfoContext(ctx, "botService init finished successfully")

	var wg syncutils.MyWaitGroup
	wg.Go(func() {
		service.Polling(ctx)
	})

	sig := <-sigChan
	blog.WarnContext(ctx, "Received signal", "signal", sig)
	cancel()
	wg.Wait()
	span.AddEvent("recieved signal, gracefully shutdown..")
	span.SetAttributes(
		attribute.String("os.signal.name", sig.String()),
	)
	span.End()
}

func (s *botService) Polling(ctx context.Context) {
	var offset int64 = 0
	for {
		// TODO: передивитися може щось подібне зробити десь ще
		select {
		case <-ctx.Done():
			blog.InfoContext(ctx, "gracefully shutdown polling process")
			return
		default:
		}

		s.pollOnce(context.Background(), &offset)
	}
}

func (s *botService) pollOnce(ctx context.Context, offset *int64) {
	ctx, span := s.observer.Tracer.Start(ctx, "BotPolling")
	defer span.End()

	updates, err := s.GetUpdates(ctx, offset)
	if err != nil {
		span.SetStatus(codes.Error, "Trouble to get updates")
		span.RecordError(err)
		blog.ErrorContext(ctx, "Trying to get updates", "error", err)

		time.Sleep(2 * time.Second)
	}

	for _, update := range updates {
		if err := s.HandleUpdate(ctx, update.Message); err != nil {
			span.SetStatus(codes.Error, "Trouble to handle update")
			span.RecordError(err)
			blog.ErrorContext(ctx, "Handling update", "error", err)
		}
		*offset = int64(update.UpdateID + 1)
	}
}
