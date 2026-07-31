package processing

import (
	"context"
	"encoding/json"

	"github.com/Nox1KCL/Arbitrage/internal/database/models"
)

type MatchResult struct {
	UserID int64
	Spread Spread
	Sub    models.Subscription
}

func Filter(ctx context.Context, channel <-chan TickerForm, payload chan<- []byte, maps *models.CachedMaps) {
	aggregator := NewAggregator(maps)
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-channel:
			if !ok {
				return
			}

			spread := Aggregation(aggregator, &data)
			if spread == nil {
				continue
			}
			matches := Match(aggregator, spread)
			if len(matches) == 0 {
				continue
			}

			bytesData, err := json.Marshal(&matches)
			if err != nil {
				return
			}
			payload <- bytesData
		}
	}
}

func Match(agg *Aggregator, spread *Spread) []*MatchResult {
	matches := []*MatchResult{}
	subs := agg.Maps.SubsBySymbol[spread.Symbol]
	for _, sub := range subs {
		if spread.Spread >= sub.MinSpreadPercent && spread.FirstVolume >= sub.MinVolume && spread.SecondVolume >= sub.MinVolume {
			if CheckAlert(agg, sub, spread) {
				matches = append(matches, &MatchResult{
					UserID: sub.TelegramChatID,
					Spread: *spread,
					Sub:    sub,
				})
				agg.LastAlerts[sub.TelegramChatID][spread.Symbol] = spread.Spread
			}
		}
	}
	return matches
}

func CheckAlert(agg *Aggregator, sub models.Subscription, spread *Spread) bool {
	if agg.LastAlerts[sub.TelegramChatID] == nil {
		agg.LastAlerts[sub.TelegramChatID] = make(map[string]float64)
	}
	lastAlert, exists := agg.LastAlerts[sub.TelegramChatID][spread.Symbol]
	if !exists {
		return true
	}
	priceChange := (spread.Spread - lastAlert) / lastAlert * 100
	return priceChange >= sub.MinPriceChangePercent
}
