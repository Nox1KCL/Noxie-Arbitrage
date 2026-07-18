package logger

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"
)

func TestHandlerConveyor(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	cfg := &LumberConfig{
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
		Compress:   false,
	}

	handler, err := HandlerConveyor(logPath, slog.LevelInfo, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handler == nil {
		t.Fatal("handler should not be nil")
	}
}

func TestGetHandler_FiltersByLevel(t *testing.T) {
	dir := t.TempDir()

	cfg := &LumberConfig{
		MaxSize:    10,
		MaxAge:     28,
		MaxBackups: 3,
		Compress:   false,
	}

	levels := map[slog.Level]string{
		slog.LevelError: filepath.Join(dir, "error.log"),
	}

	handler, err := GetHandler(cfg, levels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if handler.Enabled(context.TODO(), slog.LevelInfo) {
		t.Error("handler should NOT be enabled for Info (only Error was configured)")
	}
	if !handler.Enabled(context.TODO(), slog.LevelError) {
		t.Error("handler should be enabled for Error")
	}
}

func TestLeveledHandler_WithAttrs(t *testing.T) {
	dir := t.TempDir()

	cfg := &LumberConfig{
		MaxSize:    10,
		MaxAge:     28,
		MaxBackups: 3,
		Compress:   false,
	}

	levels := map[slog.Level]string{
		slog.LevelInfo: filepath.Join(dir, "info.log"),
	}

	base, err := GetHandler(cfg, levels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	withAttrs := base.WithAttrs([]slog.Attr{slog.String("module", "test")})
	if withAttrs == nil {
		t.Fatal("WithAttrs should not return nil")
	}
}
