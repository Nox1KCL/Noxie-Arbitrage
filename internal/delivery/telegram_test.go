package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	telemetry "github.com/Nox1KCL/Arbitrage/internal/observer"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func dummyTelemetry() (*telemetry.Observe, *deliveryMetrics) {
	tp := tracenoop.NewTracerProvider()
	mp := metricnoop.NewMeterProvider()
	obs := &telemetry.Observe{
		Tracer: tp.Tracer("test"),
		Meter:  mp.Meter("test"),
	}
	m, _ := NewDeliveryMetrics(obs.Meter)
	return obs, m
}

func TestSendTelegramMessage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"ok":true,"result":{}}`))
		}))
		defer server.Close()
		setup(t, server)

		obs, m := dummyTelemetry()
		err := sendTelegramMessage(context.Background(), 123, "hello", obs, m)
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
	})

	t.Run("business error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"ok":false,"error_code":400,"description":"chat not found"}`))
		}))
		defer server.Close()
		setup(t, server)

		obs, m := dummyTelemetry()
		err := sendTelegramMessage(context.Background(), 123, "hello", obs, m)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "chat not found") {
			t.Fatalf("expected error to contain description, got: %v", err)
		}
	})

	t.Run("http failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer server.Close()
		setup(t, server)

		obs, m := dummyTelemetry()
		err := sendTelegramMessage(context.Background(), 123, "hello", obs, m)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("token not set", func(t *testing.T) {
		t.Setenv("TELEGRAM_BOT_TOKEN", "")

		obs, m := dummyTelemetry()
		err := sendTelegramMessage(context.Background(), 123, "hello", obs, m)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "TELEGRAM_BOT_TOKEN") {
			t.Fatalf("expected error to mention env var, got: %v", err)
		}
	})

	t.Run("request payload", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/botTESTTOKEN/sendMessage" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			var body sendMessageRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decoding request body: %v", err)
			}
			if body.ChatID != 987654 {
				t.Errorf("expected chat_id 987654, got %d", body.ChatID)
			}
			if body.Text != "test text" {
				t.Errorf("expected text %q, got %q", "test text", body.Text)
			}

			w.Write([]byte(`{"ok":true,"result":{}}`))
		}))
		defer server.Close()
		setup(t, server)

		obs, m := dummyTelemetry()
		if err := sendTelegramMessage(context.Background(), 987654, "test text", obs, m); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func setup(t *testing.T, server *httptest.Server) {
	t.Helper()
	old := telegramAPI
	telegramAPI = server.URL
	t.Cleanup(func() {
		telegramAPI = old
	})
	t.Setenv("TELEGRAM_BOT_TOKEN", "TESTTOKEN")
}
