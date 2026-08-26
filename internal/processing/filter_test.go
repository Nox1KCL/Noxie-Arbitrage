package processing_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	"github.com/Nox1KCL/Arbitrage/internal/processing"
	"github.com/Nox1KCL/Arbitrage/internal/syncutils"
	"github.com/Nox1KCL/Arbitrage/internal/transport"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func newDummyTelemetry() *telemetry.Observe {
	tp := tracenoop.NewTracerProvider()
	mp := metricnoop.NewMeterProvider()
	return &telemetry.Observe{
		Tracer: tp.Tracer("test"),
		Meter:  mp.Meter("test"),
	}
}

var maps = &models.CachedMaps{
	InterestedSymbols: map[string]bool{"BTCUSDT": true},
	SubsBySymbol: map[string][]models.Subscription{
		"BTCUSDT": {
			{
				TelegramChatID:        111,
				Symbol:                "BTCUSDT",
				MinSpreadPercent:      1.0,
				MinVolume:             100.0,
				MinPriceChangePercent: 10.0,
			},
		},
	},
}

func TestCheckAlert(t *testing.T) {
	lastAlertsWith2Percent := map[int64]map[string]float64{
		111: {"BTCUSDT": 2.0},
	}

	t.Run("First message (LastAlerts is empty)", func(t *testing.T) {
		subFirstAlert := models.Subscription{
			TelegramChatID:        111,
			Symbol:                "BTCUSDT",
			MinPriceChangePercent: 20.0,
		}
		spreadFirstAlert := &processing.Spread{
			Symbol: "BTCUSDT",
			Spread: 2.0,
		}
		agg := newTestAggregator(maps)

		res := processing.CheckAlert(subFirstAlert, spreadFirstAlert, agg)
		if !res {
			t.Errorf("expected true, got false")
		}
	})

	t.Run("Spread increased highly", func(t *testing.T) {
		subSubstantialGrowth := models.Subscription{
			TelegramChatID:        111,
			Symbol:                "BTCUSDT",
			MinPriceChangePercent: 25.0,
		}
		spreadSubstantialGrowth := &processing.Spread{
			Symbol: "BTCUSDT",
			Spread: 3.0,
		}
		agg := newTestAggregator(maps, lastAlertsWith2Percent)

		res := processing.CheckAlert(subSubstantialGrowth, spreadSubstantialGrowth, agg)
		if !res {
			t.Errorf("expected true, got false")
		}
	})

	t.Run("Spread increased slightly", func(t *testing.T) {
		subMinorGrowth := models.Subscription{
			TelegramChatID:        111,
			Symbol:                "BTCUSDT",
			MinPriceChangePercent: 25.0,
		}
		spreadMinorGrowth := &processing.Spread{
			Symbol: "BTCUSDT",
			Spread: 2.2,
		}
		agg := newTestAggregator(maps, lastAlertsWith2Percent)

		res := processing.CheckAlert(subMinorGrowth, spreadMinorGrowth, agg)
		if res {
			t.Errorf("expected false, got true")
		}
	})

	t.Run("Spread went down", func(t *testing.T) {
		subSpreadDrop := models.Subscription{
			TelegramChatID:        111,
			Symbol:                "BTCUSDT",
			MinPriceChangePercent: 10.0,
		}
		spreadDrop := &processing.Spread{
			Symbol: "BTCUSDT",
			Spread: 1.5,
		}
		agg := newTestAggregator(maps, lastAlertsWith2Percent)

		res := processing.CheckAlert(subSpreadDrop, spreadDrop, agg)
		if res {
			t.Errorf("expected false, got true")
		}
	})

	t.Run("Spread hits exactly on the threshold", func(t *testing.T) {
		subExactThreshold := models.Subscription{
			TelegramChatID:        111,
			Symbol:                "BTCUSDT",
			MinPriceChangePercent: 20.0,
		}
		spreadExactThreshold := &processing.Spread{
			Symbol: "BTCUSDT",
			Spread: 2.4,
		}
		agg := newTestAggregator(maps, lastAlertsWith2Percent)

		res := processing.CheckAlert(subExactThreshold, spreadExactThreshold, agg)
		if !res {
			t.Errorf("expected true, got false")
		}
	})
}

func TestMatch(t *testing.T) {
	t.Run("No one subscribed on symbol", func(t *testing.T) {
		spreadNoSubs := &processing.Spread{
			Symbol:       "ETHUSDT",
			Spread:       2.5,
			FirstVolume:  500.0,
			SecondVolume: 500.0,
		}
		agg := newTestAggregator(maps)

		res := processing.Match(spreadNoSubs, agg)
		if len(res) != 0 {
			t.Errorf("expected 0 matches, got %v", res)
		}
	})

	t.Run("Spread is too small", func(t *testing.T) {
		spreadLow := &processing.Spread{
			Symbol:       "BTCUSDT",
			Spread:       0.1,
			FirstVolume:  500.0,
			SecondVolume: 500.0,
		}
		agg := newTestAggregator(maps)

		res := processing.Match(spreadLow, agg)

		if len(res) != 0 {
			t.Errorf("expected 0 matches, got %v", len(res))
		}
	})

	t.Run("Volume is too small", func(t *testing.T) {
		spreadLowVolume := &processing.Spread{
			Symbol:       "BTCUSDT",
			Spread:       2.0,
			FirstVolume:  90.0,
			SecondVolume: 600.0,
		}
		agg := newTestAggregator(maps)

		res := processing.Match(spreadLowVolume, agg)

		if len(res) != 0 {
			t.Errorf("expected 0 matches, got %v", len(res))
		}
	})

	t.Run("Happy path", func(t *testing.T) {
		spreadSuccess := &processing.Spread{
			Symbol:       "BTCUSDT",
			Spread:       2.5,
			FirstVolume:  200.0,
			SecondVolume: 300.0,
		}
		agg := newTestAggregator(maps)

		res := processing.Match(spreadSuccess, agg)
		if len(res) == 0 {
			t.Fatalf("expected 1 and more matches, got %v", res)
		}
		if res[0].UserID != 111 {
			t.Errorf("expected user ID, got %v", res[0].UserID)
		}
		if agg.LastAlerts[111]["BTCUSDT"] != spreadSuccess.Spread {
			t.Errorf("expected saved spread, got %v", agg.LastAlerts[111]["BTCUSDT"])
		}
	})

	t.Run("two users with different filters", func(t *testing.T) {
		multiUserMaps := &models.CachedMaps{
			SubsBySymbol: map[string][]models.Subscription{
				"BTCUSDT": {
					{TelegramChatID: 101, Symbol: "BTCUSDT", MinSpreadPercent: 1.0, MinVolume: 50.0}, // Пройде
					{TelegramChatID: 102, Symbol: "BTCUSDT", MinSpreadPercent: 5.0, MinVolume: 50.0}, // Не пройде
				},
			},
		}
		spreadForOne := &processing.Spread{
			Symbol:       "BTCUSDT",
			Spread:       2.0,
			FirstVolume:  100.0,
			SecondVolume: 100.0,
		}
		agg := newTestAggregator(multiUserMaps)

		res := processing.Match(spreadForOne, agg)
		if len(res) != 1 {
			t.Errorf("expected 1 match, got %v", res)
		}
	})
}

func TestFormMessage(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		res := processing.FormMessage([]*processing.MatchResult{})
		if len(res) != 0 {
			t.Errorf("expected 0 messages, got %v", len(res))
		}
	})

	t.Run("single match formats message correctly", func(t *testing.T) {
		matches := []*processing.MatchResult{
			{
				UserID: 123456789,
				Spread: processing.Spread{
					Symbol:       "BTCUSDT",
					BuyExchange:  "Binance",
					BuyPrice:     67100.0,
					SellExchange: "Bybit",
					SellPrice:    67200.0,
					Spread:       0.1490,
					FirstVolume:  10.5,
					SecondVolume: 20.0,
				},
			},
		}

		res := processing.FormMessage(matches)
		if len(res) != 1 {
			t.Fatalf("expected 1 message, got %v", len(res))
		}
		if res[0].TelegramUserID != 123456789 {
			t.Errorf("expected user ID 123456789, got %v", res[0].TelegramUserID)
		}

		expectedText := "Arbitrage BTCUSDT\nBuy: Binance @ 67100.0000\nSell: Bybit @ 67200.0000\nSpread: 0.1490% | Volume: 10.5000 / 20.0000"
		if res[0].Text != expectedText {
			t.Errorf("expected text:\n%v\ngot:\n%v", expectedText, res[0].Text)
		}
	})

	t.Run("multiple matches produce correct count and IDs", func(t *testing.T) {
		matches := []*processing.MatchResult{
			{UserID: 101, Spread: processing.Spread{Symbol: "BTCUSDT", BuyExchange: "Binance", SellExchange: "Bybit"}},
			{UserID: 102, Spread: processing.Spread{Symbol: "ETHUSDT", BuyExchange: "OKX", SellExchange: "Bybit"}},
		}

		res := processing.FormMessage(matches)
		if len(res) != 2 {
			t.Fatalf("expected 2 messages, got %v", len(res))
		}
		if res[0].TelegramUserID != 101 {
			t.Errorf("expected user ID 101, got %v", res[0].TelegramUserID)
		}
		if res[1].TelegramUserID != 102 {
			t.Errorf("expected user ID 102, got %v", res[1].TelegramUserID)
		}
	})
}

func TestFilter(t *testing.T) {
	wg := syncutils.MyWaitGroup{}
	obs := newDummyTelemetry()
	store := processing.NewTestSubscriptionStore(maps)

	t.Run("Cancel context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		mockInput := make(chan processing.TickerForm, 5)
		mockOutput := make(chan []*transport.FormedMessage, 5)

		cancel()
		errChan := make(chan error, 1)
		func() {
			errChan <- processing.Filter(ctx, mockInput, mockOutput, store, obs)
		}()

		res := <-errChan
		if !strings.Contains(res.Error(), "context cancel signal") {
			t.Errorf("expected error with cancel context, got %s", res)
		}
	})

	t.Run("closed channel", func(t *testing.T) {
		ctx, _ := context.WithCancel(context.Background())
		mockInput := make(chan processing.TickerForm, 5)
		mockOutput := make(chan []*transport.FormedMessage, 5)

		close(mockInput)
		errChan := make(chan error, 1)
		func() {
			errChan <- processing.Filter(ctx, mockInput, mockOutput, store, obs)
		}()

		res := <-errChan
		if !strings.Contains(res.Error(), "closed broker channel") {
			t.Errorf("expected closed broker channel error, got %s", res)
		}
	})

	t.Run("Happy path", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mockInput := make(chan processing.TickerForm, 5)
		mockOutput := make(chan []*transport.FormedMessage, 5)

		now := time.Now().UnixMilli()
		tickers := []processing.TickerForm{
			{
				ExchangeName: "Binance",
				Symbol:       "BTCUSDT",
				BestAsk:      100.0,
				BestBid:      99.0,
				Volume:       200.0,
				Timestamp:    now,
			},
			{
				ExchangeName: "Bybit",
				Symbol:       "BTCUSDT",
				BestAsk:      105.0,
				BestBid:      103.0,
				Volume:       200.0,
				Timestamp:    now,
			},
		}

		wg.Go(func() {
			_ = processing.Filter(ctx, mockInput, mockOutput, store, obs)
		})

		for _, t := range tickers {
			mockInput <- t
		}

		firstRes := <-mockOutput
		if firstRes != nil {
			t.Errorf("expected nil for single exchange, got %v", firstRes)
		}

		secondRes := <-mockOutput
		if len(secondRes) != 1 {
			t.Fatalf("expected 1 message, got %d", len(secondRes))
		}
		if secondRes[0].Text == "" {
			t.Errorf("expected non-empty message text")
		}
	})

	t.Run("Tickers without profit", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mockInput := make(chan processing.TickerForm, 5)
		mockOutput := make(chan []*transport.FormedMessage, 5)

		tickers := []processing.TickerForm{
			{
				ExchangeName: "Binance",
				Symbol:       "DOGEUSDT",
				BestAsk:      1.0,
				BestBid:      0.9,
				Volume:       500.0,
				Timestamp:    time.Now().UnixMilli(),
			},
		}

		wg.Go(func() {
			_ = processing.Filter(ctx, mockInput, mockOutput, store, obs)
		})

		for _, t := range tickers {
			mockInput <- t
		}

		msgs := <-mockOutput
		if msgs != nil {
			t.Errorf("expected empty messages, got %v", msgs)
		}
	})
}
