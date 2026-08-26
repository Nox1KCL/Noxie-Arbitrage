package processing_test

import (
	"testing"
	"time"

	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	"github.com/Nox1KCL/Arbitrage/internal/processing"
)

func newTestAggregator(maps *models.CachedMaps, lastAlerts ...map[int64]map[string]float64) *processing.Aggregator {
	alerts := make(map[int64]map[string]float64)
	if len(lastAlerts) > 0 && lastAlerts[0] != nil {
		alerts = lastAlerts[0]
	}

	return &processing.Aggregator{
		Ticks:      make(map[string]map[string]*processing.TickerForm),
		GetMaps:    func() *models.CachedMaps { return maps },
		LastAlerts: alerts,
	}
}

func TestAggregation(t *testing.T) {
	maps := &models.CachedMaps{
		InterestedSymbols: map[string]bool{"BTCUSDT": true},
		SubsBySymbol: map[string][]models.Subscription{
			"BTCUSDT": {
				{
					TelegramChatID:        123456789,
					Symbol:                "BTCUSDT",
					MinSpreadPercent:      0.01,
					MinVolume:             100,
					MinPriceChangePercent: 1.0,
				},
			},
		},
	}

	agg := newTestAggregator(maps)
	now := time.Now().UnixMilli()
	tickers := []*processing.TickerForm{
		{
			ExchangeName: "Binance",
			Symbol:       "DOGEUSDT",
			CurrentPrice: 1032.0,
			BestAsk:      1042.0,
			BestBid:      1020.0,
			Volume:       400.2,
			Timestamp:    now,
		},
		{
			ExchangeName: "Binance",
			Symbol:       "BTCUSDT",
			CurrentPrice: 67145.0,
			BestAsk:      67155.0,
			BestBid:      67150.0,
			Volume:       1200.5,
			Timestamp:    now,
		},
		{
			ExchangeName: "Bybit",
			Symbol:       "BTCUSDT",
			CurrentPrice: 67130.0,
			BestAsk:      67130.0,
			BestBid:      67125.0,
			Volume:       980.2,
			Timestamp:    now,
		},
	}

	t.Run("ignore not interested symbol", func(t *testing.T) {
		res := processing.Aggregation(agg, tickers[0])
		if res != nil {
			t.Errorf("expected nil, got %v", res)
		}
	})

	t.Run("single exchange returns nil", func(t *testing.T) {
		res := processing.Aggregation(agg, tickers[1])
		if res != nil {
			t.Errorf("expected nil, got %v", res)
		}
	})

	t.Run("two exchanges produce profitable spread", func(t *testing.T) {
		res := processing.Aggregation(agg, tickers[2])
		if res == nil {
			t.Fatal("expected spread, got nil")
		}
		if res.BuyExchange != "Bybit" || res.SellExchange != "Binance" {
			t.Errorf("unexpected exchanges: buy=%s, sell=%s", res.BuyExchange, res.SellExchange)
		}
	})

	t.Run("update data of exchange without spread", func(t *testing.T) {
		agg := newTestAggregator(maps)
		now := time.Now().UnixMilli()
		tickers := []*processing.TickerForm{
			{
				ExchangeName: "Bybit",
				Symbol:       "BTCUSDT",
				CurrentPrice: 67146.00,
				BestAsk:      67150.00,
				BestBid:      67142.00,
				Volume:       980.15,
				Timestamp:    now,
			},
			{
				ExchangeName: "Binance",
				Symbol:       "BTCUSDT",
				CurrentPrice: 67142.00,
				BestAsk:      67145.00,
				BestBid:      67140.00,
				Volume:       1250.30,
				Timestamp:    now,
			},
		}

		_ = processing.Aggregation(agg, tickers[0])

		res := processing.Aggregation(agg, tickers[1])
		if res != nil {
			t.Errorf("expected nil, got %v", res)
		}

		data := agg.Ticks["BTCUSDT"]["Binance"]
		if data.CurrentPrice != tickers[1].CurrentPrice {
			t.Errorf("expected %f, got %f", tickers[1].CurrentPrice, data.CurrentPrice)
		}
	})

	t.Run("TTL is out of terms", func(t *testing.T) {
		agg := newTestAggregator(maps)
		tickers := []*processing.TickerForm{
			{
				ExchangeName: "Bybit",
				Symbol:       "BTCUSDT",
				CurrentPrice: 67146.00,
				BestAsk:      67150.00,
				BestBid:      67142.00,
				Volume:       980.15,
				Timestamp:    time.Now().Add(-6 * time.Second).UnixMilli(),
			},
			{
				ExchangeName: "Binance",
				Symbol:       "BTCUSDT",
				CurrentPrice: 67145.0,
				BestAsk:      67155.0,
				BestBid:      67150.0,
				Volume:       1200.5,
				Timestamp:    time.Now().UnixMilli(),
			},
		}

		_ = processing.Aggregation(agg, tickers[0])
		res := processing.Aggregation(agg, tickers[1])
		if res != nil {
			t.Errorf("expected nil, got %v", res)
		}
	})

	t.Run("Three exchanges, choicing extremum", func(t *testing.T) {
		agg := newTestAggregator(maps)
		now := time.Now().UnixMilli()
		tickers := []*processing.TickerForm{
			{
				ExchangeName: "Binance",
				Symbol:       "BTCUSDT",
				CurrentPrice: 100.20,
				BestAsk:      100.00,
				BestBid:      99.50,
				Volume:       1200.50,
				Timestamp:    now,
			},
			{
				ExchangeName: "Bybit",
				Symbol:       "BTCUSDT",
				CurrentPrice: 102.50,
				BestAsk:      102.00,
				BestBid:      103.00,
				Volume:       980.20,
				Timestamp:    now,
			},
			{
				ExchangeName: "OKX",
				Symbol:       "BTCUSDT",
				CurrentPrice: 105.80,
				BestAsk:      106.50,
				BestBid:      106.00,
				Volume:       1500.30,
				Timestamp:    now,
			},
		}
		expectedSpread := (106.00 - 100.00) / 100.00 * 100

		_ = processing.Aggregation(agg, tickers[0])
		_ = processing.Aggregation(agg, tickers[1])
		res := processing.Aggregation(agg, tickers[2])

		if res == nil {
			t.Fatalf("expected spread, got %v", res)
		}
		if tickers[0].ExchangeName != res.BuyExchange || tickers[2].ExchangeName != res.SellExchange {
			t.Errorf("expected buy/sell exchange to be %s/%s, got %s/%s",
				tickers[0].ExchangeName,
				tickers[2].ExchangeName,
				res.BuyExchange,
				res.SellExchange,
			)
		}
		if res.Spread != expectedSpread {
			t.Errorf("expected spread %f, got %f", expectedSpread, res.Spread)
		}
	})
}
