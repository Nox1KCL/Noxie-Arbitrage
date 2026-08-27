package logger_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/Nox1KCL/Arbitrage/internal/logger"
)

type mockHandler struct {
	enabledResult bool
	handleErr     error
	handleCalled  bool
}

func (m *mockHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return m.enabledResult
}

func (m *mockHandler) Handle(_ context.Context, _ slog.Record) error {
	m.handleCalled = true
	return m.handleErr
}

func (m *mockHandler) WithAttrs(_ []slog.Attr) slog.Handler { return m }
func (m *mockHandler) WithGroup(_ string) slog.Handler      { return m }

func TestLeveledHandler_Enabled(t *testing.T) {
	t.Run("At least one handler enable", func(t *testing.T) {
		handlers := logger.NewHandlerEntries(
			logger.NewHandlerEntry(slog.LevelInfo, &mockHandler{enabledResult: false}),
			logger.NewHandlerEntry(slog.LevelError, &mockHandler{enabledResult: true}),
		)
		leveledHandler := logger.NewLeveledHandler(
			handlers,
		)

		res := leveledHandler.Enabled(context.Background(), slog.LevelInfo)
		if !res {
			t.Errorf("expected true, got %t", res)
		}
	})

	t.Run("Two handlers are false", func(t *testing.T) {
		handlers := logger.NewHandlerEntries(
			logger.NewHandlerEntry(slog.LevelInfo, &mockHandler{enabledResult: false}),
			logger.NewHandlerEntry(slog.LevelError, &mockHandler{enabledResult: false}),
		)
		leveledHandler := logger.NewLeveledHandler(
			handlers,
		)

		res := leveledHandler.Enabled(context.Background(), slog.LevelInfo)
		if res {
			t.Errorf("expected false, got %t", res)
		}
	})

	t.Run("Handlers are empty", func(t *testing.T) {
		handlers := logger.NewHandlerEntries()
		leveledHandler := logger.NewLeveledHandler(handlers)

		res := leveledHandler.Enabled(context.Background(), slog.LevelInfo)
		if res {
			t.Errorf("expected false, got %t", res)
		}
	})
}

func TestLeveledHandler_Handle(t *testing.T) {
	t.Run("Happy path", func(t *testing.T) {
		handlers := logger.NewHandlerEntries(
			logger.NewHandlerEntry(slog.LevelInfo, &mockHandler{enabledResult: true, handleErr: nil}),
		)
		leveledHandler := logger.NewLeveledHandler(
			handlers,
		)

		res := leveledHandler.Handle(context.Background(), slog.Record{Level: slog.LevelInfo, Message: ""})
		if res != nil {
			t.Errorf("expected nil, got %v", res)
		}
	})

	t.Run("Handler err is not nil", func(t *testing.T) {
		handlers := logger.NewHandlerEntries(
			logger.NewHandlerEntry(slog.LevelInfo, &mockHandler{enabledResult: true, handleErr: errors.New("Mock error")}),
		)
		leveledHandler := logger.NewLeveledHandler(
			handlers,
		)

		res := leveledHandler.Handle(context.Background(), slog.Record{Time: time.Now(), Level: slog.LevelInfo, Message: "Mock error", PC: 0})
		if res == nil {
			t.Errorf("expected non-nil error, got nil")
		}
	})

	t.Run("Handler is disable", func(t *testing.T) {
		handlers := logger.NewHandlerEntries(
			logger.NewHandlerEntry(slog.LevelInfo, &mockHandler{enabledResult: false}),
		)
		leveledHandler := logger.NewLeveledHandler(
			handlers,
		)

		res := leveledHandler.Handle(context.Background(), slog.Record{Level: slog.LevelInfo, Message: "Mock error"})
		if res != nil {
			t.Errorf("expected nil, got %s", res.Error())
		}
	})
}

func TestLeveledHandler_WithAttrs(t *testing.T) {
	handlers := logger.NewHandlerEntries(
		logger.NewHandlerEntry(slog.LevelInfo, &mockHandler{}),
		logger.NewHandlerEntry(slog.LevelError, &mockHandler{}),
	)
	h := logger.NewLeveledHandler(handlers)

	newHandler := h.WithAttrs([]slog.Attr{slog.String("key", "value")})

	lh, ok := newHandler.(*logger.LeveledHandler)
	if !ok {
		t.Fatalf("expected *LeveledHandler, got %T", newHandler)
	}
	if len(lh.Handlers()) != len(handlers) {
		t.Errorf("expected %d handlers, got %d", len(handlers), len(lh.Handlers()))
	}
}
