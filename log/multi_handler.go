package log

import (
	"context"
	"errors"
	"log/slog"
)

// MultiHandler is a slog.Handler that dispatches log records to multiple handlers.
// It combines the behavior of multiple handlers into a single handler. When a log
// record is handled, it is sent to all the registered handlers. If any handler
// returns an error, the errors are joined together.
type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{
		handlers: handlers,
	}
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	var err error
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, record.Level) {
			err = errors.Join(err, handler.Handle(ctx, record))
		}
	}
	return err
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := &MultiHandler{}
	for _, handler := range h.handlers {
		h2.handlers = append(h2.handlers, handler.WithAttrs(attrs))
	}
	return h2
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	h2 := &MultiHandler{}
	for _, handler := range h.handlers {
		h2.handlers = append(h2.handlers, handler.WithGroup(name))
	}
	return h2
}
