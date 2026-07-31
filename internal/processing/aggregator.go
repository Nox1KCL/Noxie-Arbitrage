package processing

import (
	"time"

	"github.com/Nox1KCL/Arbitrage/internal/database/models"
)

type Aggregator struct {
	Ticks map[string]map[string]*TickerForm // Symbol / Exchanges
	Maps  *models.CachedMaps
}

type Spread struct {
	BuyExchange  string
	SellExchange string
	BuyPrice     float64
	SellPrice    float64
	Spread       float64
}

func NewAggregator(maps *models.CachedMaps) *Aggregator {
	return &Aggregator{
		Ticks: make(map[string]map[string]*TickerForm),
		Maps:  maps,
	}
}

func Aggregation(agg *Aggregator, t *TickerForm) *Spread {
	_, exists := agg.Maps.InterestedSymbols[t.Symbol]
	if !exists {
		return nil
	}
	if agg.Ticks[t.Symbol] == nil {
		agg.Ticks[t.Symbol] = make(map[string]*TickerForm)
	}
	agg.Ticks[t.Symbol][t.ExchangeName] = t
	if len(agg.Ticks[t.Symbol]) < 2 {
		return nil
	}

	agg.TTLChecker()
	spread := agg.Arbitrage(t.Symbol)
	if spread != nil {
		return spread
	}
	return nil
}

func (agg *Aggregator) TTLChecker() {
	for _, batch := range agg.Ticks {
		for exchange, t := range batch {
			if time.Since(time.UnixMilli(t.Timestamp)) >= 5*time.Second {
				delete(batch, exchange)
			}
		}
	}

}

func (agg *Aggregator) Arbitrage(symbol string) *Spread {
	var (
		minAskTicker TickerForm
		maxBidTicker TickerForm
		first        = true
	)
	exchanges := agg.Ticks[symbol]
	for _, t := range exchanges {
		if first {
			minAskTicker = *t
			maxBidTicker = *t
			first = false
			continue
		}
		if t.BestAsk < minAskTicker.BestAsk {
			minAskTicker = *t
		}
		if t.BestBid > maxBidTicker.BestBid {
			maxBidTicker = *t
		}
	}
	if maxBidTicker.BestBid > minAskTicker.BestAsk && minAskTicker.ExchangeName != maxBidTicker.ExchangeName {
		buyPrice := minAskTicker.BestAsk
		sellPrice := maxBidTicker.BestBid
		spread := ((sellPrice - buyPrice) / buyPrice) * 100
		return &Spread{
			BuyExchange:  minAskTicker.ExchangeName,
			SellExchange: maxBidTicker.ExchangeName,
			BuyPrice:     buyPrice,
			SellPrice:    sellPrice,
			Spread:       spread,
		}
	}
	return nil
}
