package logger

import "log/slog"

func NewLeveledHandler(handlers []handlerEntry) *LeveledHandler {
	return &LeveledHandler{
		handlers: handlers,
	}
}

func NewHandlerEntry(level slog.Level, handler slog.Handler) handlerEntry {
	return handlerEntry{
		level:   level,
		handler: handler,
	}
}

func NewHandlerEntries(entries ...handlerEntry) []handlerEntry {
	return entries
}

func (h *LeveledHandler) Handlers() []handlerEntry {
	return h.handlers
}
