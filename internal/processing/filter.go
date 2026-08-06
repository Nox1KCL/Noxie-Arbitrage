package processing

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/Nox1KCL/Arbitrage/internal/database"
	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	"github.com/Nox1KCL/Arbitrage/internal/transport"
	"gorm.io/gorm"
)

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

func Filter(ctx context.Context, channel <-chan TickerForm, payload chan<- []*transport.FormedMessage, store *SubscriptionStore) {
	aggregator := NewAggregator(store)
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
	subs := agg.GetMaps().SubsBySymbol[spread.Symbol]
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
