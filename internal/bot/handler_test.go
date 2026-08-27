package bot_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nox1KCL/Arbitrage/internal/bot"
)

func TestHandleUpdate(t *testing.T) {
	var sentText string
	var sentChatID int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req bot.SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		sentText = req.Text
		sentChatID = req.ChatID

		w.Write([]byte(`{"ok": true, "result": {}}`))
	}))
	defer server.Close()

	cleanup := bot.SetTelegramAPI(server.URL)
	defer cleanup()

	obs := bot.NewTestDummyTelemetry()
	client := &bot.NewTestMockProcessingClient{}
	service, _ := bot.NewBotService(obs, nil, "TEST_TOKEN", client)

	t.Run("Command /start", func(t *testing.T) {
		sentText = ""
		cmd := "/start@Nox"
		msg := bot.NewTestMessage(
			cmd,
			bot.NewTestChat(123),
			bot.NewTestEntities("bot_command", 3, len(cmd)),
		)
		err := service.HandleUpdate(context.Background(), msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sentChatID != 123 {
			t.Errorf("expected chat ID 123, got %d", sentChatID)
		}
		if !strings.Contains(sentText, "Noxie Arbitrage Bot") {
			t.Errorf("expected welcome message, got %q", sentText)
		}
	})

	t.Run("Unknown command (default)", func(t *testing.T) {
		sentText = ""

		cmd := "/unknown@Nox"
		msg := bot.NewTestMessage(
			cmd,
			bot.NewTestChat(123),
			bot.NewTestEntities("bot_command", 3, len(cmd)),
		)
		err := service.HandleUpdate(context.Background(), msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sentChatID != 123 {
			t.Errorf("expected chat ID 123, got %d", sentChatID)
		}
		if !strings.Contains(sentText, "Sorry! I don't know that command.") {
			t.Errorf("expected default message, got %q", sentText)
		}
	})

	t.Run("No command in message", func(t *testing.T) {
		sentText = ""

		cmd := "Nox"
		msg := bot.NewTestMessage(
			cmd,
			bot.NewTestChat(123),
			bot.NewTestEntities("no_command", 3, len(cmd)),
		)
		err := service.HandleUpdate(context.Background(), msg)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}
