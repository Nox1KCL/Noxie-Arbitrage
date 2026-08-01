package processing

import (
	"context"
	"fmt"

	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	"github.com/Nox1KCL/Arbitrage/internal/transport"
)

type MatchResult struct {
	UserID int64
	Spread Spread
	Sub    models.Subscription
}

func Filter(ctx context.Context, channel <-chan TickerForm, payload chan<- []*transport.FormedMessage, maps *models.CachedMaps) {
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
			msgs := FormMessage(matches)
			payload <- msgs
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

func FormMessage(matches []*MatchResult) []*transport.FormedMessage {
	var msgs = []*transport.FormedMessage{}
	for _, m := range matches {
		s := m.Spread
		id := m.UserID

		text := fmt.Sprintf(
			"Арбітраж %s\nКупити: %s @ %.4f\nПродати: %s @ %.4f\nСпред: %.4f%% | Обсяг: %.4f / %.4f",
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
