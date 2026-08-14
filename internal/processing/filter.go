package processing

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/Nox1KCL/Arbitrage/internal/database"
	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	"github.com/Nox1KCL/Arbitrage/internal/transport"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"gorm.io/gorm"
)

var plog = slog.With("service", "processing")

type deliveryMetrics struct {
	sentMessages metric.Int64Counter
}

func NewDeliveryMetrics(meter metric.Meter) (*deliveryMetrics, error) {
	sentMessages, err := meter.Int64Counter(
		"sent_messages",
		metric.WithDescription("Total count of sent messages"),
	)
	if err != nil {
		err := fmt.Errorf("sentMessages init failed: %w", err)
		return nil, err
	}

	return &deliveryMetrics{
		sentMessages: sentMessages,
	}, nil
}

type deliveryService struct {
	metrics    *deliveryMetrics
	observer   *telemetry.Observe
	aggregator *Aggregator
}

func NewDeliveryService(obs *telemetry.Observe, agg *Aggregator) (*deliveryService, error) {
	metrics, err := NewDeliveryMetrics(obs.Meter)
	if err != nil {
		err := fmt.Errorf("getting metrics: %w", err)
		return nil, err
	}
	return &deliveryService{
		metrics:    metrics,
		observer:   obs,
		aggregator: agg,
	}, nil
}


type MatchResult struct {
	UserID int64
	Spread Spread
	Sub    models.Subscription
}

type SubscriptionStore struct {
	ptr atomic.Pointer[models.CachedMaps]
}

func (s *SubscriptionStore) Get() *models.CachedMaps {
	return s.ptr.Load()
}

func (s *SubscriptionStore) Reload(db *gorm.DB) error {
	maps, err := database.LoadSubscriptions(db, []*models.Subscription{})
	if err != nil {
		return err
	}
	s.ptr.Store(maps)
	return nil
}

func Filter(ctx context.Context, channel <-chan TickerForm, payload chan<- []*transport.FormedMessage, store *SubscriptionStore, obs *telemetry.Observe) error {
	aggregator := NewAggregator(store)

	service, err := NewDeliveryService(obs, aggregator)
	if err != nil {
		return err
	}

	_, span := service.observer.Tracer.Start(ctx, "FilterProcessing")
	defer span.End()
	span.AddEvent("Starting filter processing")

	for {
		select {
		case <-ctx.Done():
			span.SetStatus(codes.Error, "cancel context")
			return fmt.Errorf("context cancel signal")
		case data, ok := <-channel:
			if !ok {
				err := fmt.Errorf("closed broker channel: %t", ok)
				span.SetStatus(codes.Error, "closed channel")
				span.RecordError(err)

				return err
			}

			msgs, err := service.msgsProcessing(context.Background(), &data)
			if err != nil {
				plog.InfoContext(ctx, "aggregation cycle error", "error", err)
				continue
			}

			payload <- msgs
		}
	}
}

func (s *deliveryService) msgsProcessing(ctx context.Context, data *TickerForm) ([]*transport.FormedMessage, error) {
	_, span := s.observer.Tracer.Start(ctx, "AggregationCycle")
	defer span.End()

	spread := Aggregation(s.aggregator, data)
	if spread == nil {
		span.AddEvent("No spread value")
		return nil, nil
	}
	matches := s.Match(spread)
	if len(matches) == 0 {
		span.AddEvent("Matches length is zero")
		return nil, nil
	}

	msgs := FormMessage(matches)
	return msgs, nil
}

func (s *deliveryService) Match(spread *Spread) []*MatchResult {
	matches := []*MatchResult{}
	subs := s.aggregator.GetMaps().SubsBySymbol[spread.Symbol]
	for _, sub := range subs {
		if spread.Spread >= sub.MinSpreadPercent && spread.FirstVolume >= sub.MinVolume && spread.SecondVolume >= sub.MinVolume {
			if s.CheckAlert(sub, spread) {
				matches = append(matches, &MatchResult{
					UserID: sub.TelegramChatID,
					Spread: *spread,
					Sub:    sub,
				})
				s.aggregator.LastAlerts[sub.TelegramChatID][spread.Symbol] = spread.Spread
			}
		}
	}
	return matches
}

func (s *deliveryService) CheckAlert(sub models.Subscription, spread *Spread) bool {
	if s.aggregator.LastAlerts[sub.TelegramChatID] == nil {
		s.aggregator.LastAlerts[sub.TelegramChatID] = make(map[string]float64)
	}
	lastAlert, exists := s.aggregator.LastAlerts[sub.TelegramChatID][spread.Symbol]
	if !exists {
		return true
	}
	priceChange := (spread.Spread - lastAlert) / lastAlert * 100
	return priceChange >= sub.MinPriceChangePercent
}

func FormMessage(matches []*MatchResult) []*transport.FormedMessage {
	msgs := []*transport.FormedMessage{}
	for _, m := range matches {
		s := m.Spread
		id := m.UserID

		text := fmt.Sprintf(
			"Arbitrage %s\nBuy: %s @ %.4f\nSell: %s @ %.4f\nSpread: %.4f%% | Volume: %.4f / %.4f",
			s.Symbol,
			s.BuyExchange, s.BuyPrice,
			s.SellExchange, s.SellPrice,
			s.Spread, s.FirstVolume, s.SecondVolume,
		)
		msgs = append(msgs, &transport.FormedMessage{
			TelegramUserID: id,
			Text:           text,
		})
	}
	return msgs
}
