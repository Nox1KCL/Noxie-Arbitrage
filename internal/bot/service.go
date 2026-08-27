package bot

import (
	"fmt"
	"log/slog"

	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	pb "github.com/Nox1KCL/Arbitrage/internal/transport/proto"
	"go.opentelemetry.io/otel/metric"
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
	Observer *telemetry.Observe
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
		Observer: obs,
		metrics:  m,
		db:       db,
		token:    token,
		client:   client,
	}, nil
}
