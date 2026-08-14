package processing

import (
	"context"
	"encoding/json"

	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/codes"
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

func Scanner(ctx context.Context, msgs <-chan amqp091.Delivery, channel chan<- TickerForm, obs *telemetry.Observe) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-msgs:
			if !ok {
				return
			}
			ticker := gatherProcessing(ctx, data, obs)
			if ticker == (TickerForm{}) {
				continue
			}
			channel <- ticker
		}
	}
}

func gatherProcessing(ctx context.Context, data amqp091.Delivery, obs *telemetry.Observe) TickerForm {
	_, span := obs.Tracer.Start(ctx, "GatherMessage")
	defer span.End()

	var ticker TickerForm
	if err := json.Unmarshal(data.Body, &ticker); err != nil {
		span.SetStatus(codes.Error, "failed to ummarshal message")
		span.RecordError(err)

		_ = data.Nack(false, false)
		return TickerForm{}
	}
	_ = data.Ack(false)
	return ticker
}
