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

func TestPolling(t *testing.T) {
	token := "TEST_TOKEN"
	service := bot.NewTestBotService(t, token, func(chatID int64, text string) {})

	t.Run("cancel context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		errChan := make(chan error, 1)
		go func() {
			errChan <- service.Polling(ctx)
		}()

		err := <-errChan
		if !strings.Contains(err.Error(), "context cancel signal") {
			t.Errorf("expected context cancel signal, got %v", err.Error())
		}
	})
}

func TestPollOnce(t *testing.T) {
	t.Run("Happy path", func(t *testing.T) {
		var sentText string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "getUpdates") {
				w.Write([]byte(`{
                           "ok": true,
                           "result": [
                               {
                                   "update_id": 100,
                                   "message": {
                                       "text": "/start",
                                       "chat": {"id": 12345},
                                       "entities": [{"type": "bot_command", "offset": 0, "length": 6}]
                                   }
                               }
                           ]
                       }`))
				return
			}
			if strings.Contains(r.URL.Path, "sendMessage") {
				var req bot.SendMessageRequest
				_ = json.NewDecoder(r.Body).Decode(&req)
				sentText = req.Text
				w.Write([]byte(`{"ok": true, "result": {}}`))
				return
			}
		}))
		defer server.Close()

		cleanup := bot.SetTelegramAPI(server.URL)
		defer cleanup()

		obs := bot.NewTestDummyTelemetry()
		client := &bot.NewTestMockProcessingClient{}
		service, _ := bot.NewBotService(obs, nil, "TEST_TOKEN", client)

		var offset int64 = 0
		service.PollOnce(context.Background(), &offset)

		if offset != 101 {
			t.Errorf("expected offset 101, got %d", offset)
		}
		if !strings.Contains(sentText, "Noxie Arbitrage Bot") {
			t.Errorf("expected welcome message, got %q", sentText)
		}
	})
}
