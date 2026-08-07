package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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

func GetUpdates(ctx context.Context, token string, offset int64, m *botMetrics) ([]Update, error) {
	url := fmt.Sprintf("%s/bot%s/getUpdates?offset=%d&timeout=30", telegramAPI, token, offset)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		m.requestsErrors.Add(ctx, 1)
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		m.clientErrors.Add(ctx, 1)
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	var decodedResp getUpdatesResponse
	err = json.NewDecoder(resp.Body).Decode(&decodedResp)
	if err != nil {
		return nil, fmt.Errorf("decoding req: %w", err)
	}

	if !decodedResp.Ok {
		return nil, fmt.Errorf("telegram error: %t | %s", decodedResp.Ok, decodedResp.Description)
	}
	return decodedResp.Result, nil
}

func sendMessage(ctx context.Context, token string, chatID int64, text string, m *botMetrics) error {
	if token == "" {
		return fmt.Errorf("TELEGRAM_BOT_API is not set")
	}

	body, err := json.Marshal(sendMessageRequest{ChatID: chatID, Text: text})
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", telegramAPI, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		m.requestsErrors.Add(ctx, 1)
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		m.clientErrors.Add(ctx, 1)
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	var tgResp telegramResponse
	if err := json.Unmarshal(respBody, &tgResp); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}

	if !tgResp.Ok {
		return fmt.Errorf("telegram error: %s", tgResp.Description)
	}

	return nil
}
