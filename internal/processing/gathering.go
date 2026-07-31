package processing

import (
	"context"
	"encoding/json"

	"github.com/rabbitmq/amqp091-go"
)

type TickerForm struct {
	ExchangeName string  `json:"exchangeName"`
	Symbol       string  `json:"symbol"`
	CurrentPrice float64 `json:"currentPrice"`
	BestAsk      float64 `json:"bestAsk"`
	BestBid      float64 `json:"bestBid"`
	Volume       float64 `json:"volume"`
	Timestamp    int64   `json:"timestamp"`
}

func Scanner(ctx context.Context, msgs <-chan amqp091.Delivery, channel chan<- TickerForm) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-msgs:
			if !ok {
				return
			}
			var ticker TickerForm
			if err := json.Unmarshal(data.Body, &ticker); err != nil {
				data.Nack(false, false)
				continue
			}
			data.Ack(false)
			channel <- ticker
		}
	}
}
