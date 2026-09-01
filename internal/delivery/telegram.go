package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

var telegramAPI = "https://api.telegram.org"

type sendMessageRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

func sendTelegramMessage(ctx context.Context, chatID int64, text string, obs *telemetry.Observe, m *deliveryMetrics) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		m.histogram.Record(ctx, duration, metric.WithAttributes(
			attribute.String("stage", "sendTelegramMessage"),
		))
	}()

	ctx, span := obs.Tracer.Start(ctx, "SendingTelegramMessage")
	defer span.End()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		span.SetStatus(codes.Error, "telegram bot token is missing")

		return fmt.Errorf("TELEGRAM_BOT_API is not set")
	}

	body, err := json.Marshal(sendMessageRequest{ChatID: chatID, Text: text})
	if err != nil {
		span.SetStatus(codes.Error, "could not marshal request")
		span.RecordError(err)

		return fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", telegramAPI, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		span.SetStatus(codes.Error, "could not form request")
		span.RecordError(err)

		m.counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("stage", "sendTelegramMessage"),
			attribute.String("type", "request.create.error"),
		))

		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		span.SetStatus(codes.Error, "could not perform request")
		span.RecordError(err)

		m.counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("stage", "sendTelegramMessage"),
			attribute.String("type", "request.perform.error"),
		))

		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		span.SetStatus(codes.Error, "could not read response")
		span.RecordError(err)

		return fmt.Errorf("reading response: %w", err)
	}

	var tgResp telegramResponse
	if err := json.Unmarshal(respBody, &tgResp); err != nil {
		span.SetStatus(codes.Error, "could not unmarshal telegram response")
		span.RecordError(err)

		return fmt.Errorf("unmarshaling response: %w", err)
	}

	if !tgResp.OK {
		span.AddEvent("telegram !ok result")
		return fmt.Errorf("telegram error: %s", tgResp.Description)
	}

	return nil
}
