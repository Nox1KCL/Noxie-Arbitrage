package processing_test

import (
	"context"
	"testing"
	"time"

	"github.com/Nox1KCL/Arbitrage/internal/processing"
	"github.com/rabbitmq/amqp091-go"
)

func TestScanner(t *testing.T) {
	obs := newDummyTelemetry()

	t.Run("valid message is parsed and sent to channel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		input := make(chan amqp091.Delivery, 5)
		output := make(chan processing.TickerForm, 5)

		go processing.Scanner(ctx, input, output, obs)

		input <- amqp091.Delivery{
			Body: []byte(`{
				"exchangeName": "Binance",
				"symbol": "BTCUSDT",
				"currentPrice": 67000.0,
				"bestAsk": 67010.0,
				"bestBid": 66990.0,
				"volume": 150.0,
				"timestamp": 1700000000
			}`),
		}

		res := <-output
		if res.ExchangeName != "Binance" || res.Symbol != "BTCUSDT" {
			t.Errorf("unexpected ticker: %+v", res)
		}
		if res.CurrentPrice != 67000.0 || res.Volume != 150.0 {
			t.Errorf("unexpected price/volume: %+v", res)
		}
	})

	t.Run("invalid json is skipped", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		input := make(chan amqp091.Delivery, 5)
		output := make(chan processing.TickerForm, 5)

		go processing.Scanner(ctx, input, output, obs)

		input <- amqp091.Delivery{
			Body: []byte(`invalid json`),
		}

		select {
		case res := <-output:
			t.Fatalf("expected no ticker from invalid json, got %+v", res)
		case <-time.After(50 * time.Millisecond):
		}
	})

	t.Run("context cancel terminates scanner", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		input := make(chan amqp091.Delivery, 5)
		output := make(chan processing.TickerForm, 5)

		done := make(chan struct{})
		go func() {
			processing.Scanner(ctx, input, output, obs)
			close(done)
		}()

		cancel()

		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("scanner did not stop on context cancellation")
		}
	})

	t.Run("closed channel terminates scanner", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		input := make(chan amqp091.Delivery, 5)
		output := make(chan processing.TickerForm, 5)

		done := make(chan struct{})
		go func() {
			processing.Scanner(ctx, input, output, obs)
			close(done)
		}()

		close(input)

		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("scanner did not stop on channel close")
		}
	})
}
