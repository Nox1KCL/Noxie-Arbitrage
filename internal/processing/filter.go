package processing

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/Nox1KCL/Arbitrage/internal/database"
	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	"github.com/Nox1KCL/Arbitrage/internal/transport"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"gorm.io/gorm"
)

var plog = slog.With("service", "processing")

type processingMetrics struct {
	counter       metric.Int64Counter
	histogram     metric.Float64Histogram
}

func NewProcessingMetrics(meter metric.Meter) (*processingMetrics, error) {
	counter, err := meter.Int64Counter(
		"counter",
		metric.WithDescription("for counting messages, errors"),
	)
	if err != nil {
		err := fmt.Errorf("counter init failed: %w", err)
		return nil, err
	}

	histogram, err := meter.Float64Histogram(
		"histogram",
		metric.WithDescription("for calculating time of requests, filtration"),
	)
	if err != nil {
		err := fmt.Errorf("histogram init failed: %w", err)
		return nil, err
	}

	return &processingMetrics{
		counter:       counter,
		histogram:     histogram,
	}, nil
}

type processingService struct {
	metrics    *processingMetrics
	observer   *telemetry.Observe
	aggregator *Aggregator
}

func NewProcessingService(obs *telemetry.Observe, agg *Aggregator) (*processingService, error) {
	metrics, err := NewProcessingMetrics(obs.Meter)
	if err != nil {
		err := fmt.Errorf("getting metrics: %w", err)
		return nil, err
	}
	return &processingService{
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

	service, err := NewProcessingService(obs, aggregator)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancel signal")
		case data, ok := <-channel:
			if !ok {
				err := fmt.Errorf("closed broker channel: %t", ok)
				return err
			}
			msgs, err := service.msgsProcessing(context.Background(), &data)
			if err != nil {
				plog.InfoContext(ctx, "aggregation cycle error", "error", err)
				continue
			}

			service.metrics.counter.Add(ctx, 1, metric.WithAttributes(
				attribute.String("stage", "filtration"),
				attribute.String("type", "sent.data"),
			))

			payload <- msgs
		}
	}
}

func (s *processingService) msgsProcessing(ctx context.Context, data *TickerForm) ([]*transport.FormedMessage, error) {
	ctx, span := s.observer.Tracer.Start(ctx, "AggregationCycle")
	defer span.End()

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		s.metrics.histogram.Record(ctx, duration, metric.WithAttributes(
			attribute.String("stage", "aggregation"),
		))
	}()

	spread := Aggregation(s.aggregator, data)
	if spread == nil {
		span.AddEvent("No spread value")
		s.metrics.counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("stage", "aggregation.spread"),
			attribute.String("type", "no.valuable.errors"),
		))

		return nil, nil
	}
	matches := s.Match(spread)
	if len(matches) == 0 {
		span.AddEvent("Matches length is zero")
		s.metrics.counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("stage", "aggregation.matches"),
			attribute.String("type", "no.valuable.errors"),
		))

		return nil, nil
	}

	msgs := FormMessage(matches)
	return msgs, nil
}

func (s *processingService) Match(spread *Spread) []*MatchResult {
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

func (s *processingService) CheckAlert(sub models.Subscription, spread *Spread) bool {
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
