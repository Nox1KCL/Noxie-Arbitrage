package bot_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Nox1KCL/Arbitrage/internal/bot"
)

func TestSendMessage(t *testing.T) {
	t.Run("Without bot token", func(t *testing.T) {
		service := bot.NewTestBotService(t, "", nil)
		err := service.SendMessage(context.Background(), 123, "hello")
		if err == nil || !strings.Contains(err.Error(), "TELEGRAM_BOT_API is not set") {
			t.Errorf("expected token not set error, got %v", err)
		}
	})

	t.Run("Happy path", func(t *testing.T) {
		var capturedChatID int64
		var capturedText string

		service := bot.NewTestBotService(t, "TEST_TOKEN", func(chatID int64, text string) {
			capturedChatID = chatID
			capturedText = text
		})

		err := service.SendMessage(context.Background(), 98765, "Arbitrage alert!")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedChatID != 98765 {
			t.Errorf("expected chatID 98765, got %d", capturedChatID)
		}
		if capturedText != "Arbitrage alert!" {
			t.Errorf("expected text 'Arbitrage alert!', got %q", capturedText)
		}
	})
}
