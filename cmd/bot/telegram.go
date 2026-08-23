package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Update struct {
	UpdateID int64   `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	Text     string   `json:"text"`
	Chat     Chat     `json:"chat"`
	Entities []Entity `json:"entities,omitempty"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type Entity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

type sendMessageRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type telegramResponse struct {
	Ok          bool   `json:"ok"`
	Description string `json:"description"`
}

type getUpdatesResponse struct {
	Ok          bool     `json:"ok"`
	Description string   `json:"description"`
	Result      []Update `json:"result,omitempty"`
}

var telegramAPI = "https://api.telegram.org"

func (s *botService) GetUpdates(ctx context.Context, offset *int64) ([]Update, error) {
	childCtx, span := s.observer.Tracer.Start(ctx, "GetUpdates")
	defer span.End()

	url := fmt.Sprintf("%s/bot%s/getUpdates?offset=%d&timeout=30", telegramAPI, s.token, *offset)
	req, err := http.NewRequestWithContext(childCtx, "GET", url, nil)
	if err != nil {
		span.SetStatus(codes.Error, "getUpdates request")
		span.RecordError(err)

		s.metrics.requestsErrors.Add(childCtx, 1)
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		span.SetStatus(codes.Error, "getUpdates client")
		span.RecordError(err)

		s.metrics.clientErrors.Add(childCtx, 1)
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	var decodedResp getUpdatesResponse
	err = json.NewDecoder(resp.Body).Decode(&decodedResp)
	if err != nil {
		span.SetStatus(codes.Error, "could not decode response")
		span.RecordError(err)

		return nil, fmt.Errorf("decoding req: %w", err)
	}

	if !decodedResp.Ok {
		err := fmt.Errorf("telegram error: %t | %s", decodedResp.Ok, decodedResp.Description)
		span.SetStatus(codes.Error, "telegram !ok response getUpdates")
		span.RecordError(err)

		return nil, fmt.Errorf("telegram error: %t | %s", decodedResp.Ok, decodedResp.Description)
	}
	return decodedResp.Result, nil
}

func (s *botService) sendMessage(ctx context.Context, id int64, text string) error {
	if s.token == "" {
		return fmt.Errorf("TELEGRAM_BOT_API is not set")
	}
	childCtx, span := s.observer.Tracer.Start(ctx, "SendMessage")
	defer span.End()

	body, err := json.Marshal(sendMessageRequest{ChatID: id, Text: text})
	if err != nil {
		span.SetStatus(codes.Error, "could not marshal request")
		span.RecordError(err, trace.WithAttributes(
			attribute.Int("user.id", int(id)),
		))

		return fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", telegramAPI, s.token)
	req, err := http.NewRequestWithContext(
		childCtx,
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		span.SetStatus(codes.Error, "could not create request")
		span.RecordError(err, trace.WithAttributes(
			attribute.Int("user.id", int(id)),
		))

		s.metrics.requestsErrors.Add(childCtx, 1)
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		span.SetStatus(codes.Error, "could not send request")
		span.RecordError(err, trace.WithAttributes(
			attribute.Int("user.id", int(id)),
		))

		s.metrics.clientErrors.Add(childCtx, 1)
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		span.SetStatus(codes.Error, "could not read response")
		span.RecordError(err, trace.WithAttributes(
			attribute.Int("user.id", int(id)),
		))

		return fmt.Errorf("reading response: %w", err)
	}

	var tgResp telegramResponse
	if err := json.Unmarshal(respBody, &tgResp); err != nil {
		span.SetStatus(codes.Error, "could not unmarshal response")
		span.RecordError(err, trace.WithAttributes(
			attribute.Int("user.id", int(id)),
		))

		return fmt.Errorf("unmarshaling response: %w", err)
	}

	if !tgResp.Ok {
		err := fmt.Errorf("telegram error: %s", tgResp.Description)
		span.SetStatus(codes.Error, "telegram !ok response sendMessage")
		span.RecordError(err, trace.WithAttributes(
			attribute.Int("user.id", int(id)),
		))

		return fmt.Errorf("telegram error: %s", tgResp.Description)
	}

	return nil
}
