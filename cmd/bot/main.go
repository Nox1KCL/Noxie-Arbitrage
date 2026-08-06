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
	tgRequestsCount     metric.Int64Counter
}

func newBotMetrics(meter metric.Meter) (*botMetrics, error) {
	sentMessages, err := meter.Int64Counter(
		"sent_messages",
		metric.WithDescription("Total count of sent messages"),
	)
	if err != nil {
		return nil, err
	}

	dbErrors, err := meter.Int64Counter(
		"db_errors_total",
		metric.WithDescription("Total count of db errors"),
	)
	if err != nil {
		return nil, err
	}

	activeSubscriptions, err := meter.Int64UpDownCounter(
		"active_subscriptions",
		metric.WithDescription("Total count of active subscription"),
	)
	if err != nil {
		return nil, err
	}

	tgRequestsCount, err := meter.Int64Counter(
		"telegram_requests_count",
		metric.WithDescription("Total count of telegram requests count"),
	)
	if err != nil {
		return nil, err
	}

	return &botMetrics{
		sentMessages:        sentMessages,
		dbErrors:            dbErrors,
		activeSubscriptions: activeSubscriptions,
		tgRequestsCount:     tgRequestsCount,
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
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	token := os.Getenv("TELEGRAM_BOT_API")
	if token == "" {
		slog.Error("TELEGRAM_BOT_API is not set")
		os.Exit(1)
	}

	conn, err := grpc.NewClient("processing-go:50050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		blog.Error("Could not connect to client", "error", err)
		return
	}
	defer conn.Close()

	client := pb.NewProcessingServiceClient(conn)

	db, err := database.Connect()
	if err != nil {
		slog.Error("Could not connect to db", "error", err)
		os.Exit(1)
	}
	if err := db.AutoMigrate(&models.Subscription{}); err != nil {
		blog.Error("Could not create a table", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cfg, err := config.GetConfig("config.toml")
	if err != nil {
		blog.Error("Could not load config", "error", err)
		os.Exit(1)
	}
	shutdown, observer, err := telemetry.NewTelemetry(&cfg.LumberConfig)
	if err != nil {
		blog.Error("Could not get observer", "error", err)
		os.Exit(1)
	}
	defer shutdown(ctx)

	service, err := NewBotService(observer, db, token, client)
	if err != nil {
		blog.Error("Could not get bot service", "error", err)
		os.Exit(1)
	}

	var wg syncutils.MyWaitGroup
	wg.Go(func() {
		Polling(ctx, service)
	})

	sig := <-sigChan
	slog.Warn("Received signal", "signal", sig)
	cancel()
	wg.Wait()
}

func Polling(ctx context.Context, service *botService) {
	var offset int64 = 0
	for {
		// TODO: передивитися може щось подібне зробити десь ще
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, err := GetUpdates(ctx, service.token, offset)
		if err != nil {
			blog.Error("Trying to get updates", "error", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, update := range updates {
			if err := HandleUpdate(ctx, service, update.Message); err != nil {
				blog.Error("Handling update", "error", err)
			}
			offset = update.UpdateID + 1
		}
	}
}
