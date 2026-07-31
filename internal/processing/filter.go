package processing

import (
	"context"
	"encoding/json"

	"github.com/Nox1KCL/Arbitrage/internal/database/models"
)

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
			bytesData, err := json.Marshal(&spread)
			if err != nil {
				return
			}
			payload <- bytesData
		}
	}
}
