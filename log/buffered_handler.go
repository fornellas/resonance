package log

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

type handleCall struct {
	handler slog.Handler
	context context.Context
	record  slog.Record
}

func (h *handleCall) flush() error {
	return h.handler.Handle(h.context, h.record)
}

type sharedBuffer struct {
	mu          sync.Mutex
	handleCalls []handleCall
}

func (s *sharedBuffer) appendHandleCall(handler slog.Handler, ctx context.Context, record slog.Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handleCalls = append(s.handleCalls, handleCall{
		handler: handler,
		context: ctx,
		record:  record.Clone(),
	})
}

func (s *sharedBuffer) flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var errs error
	for _, handleCall := range s.handleCalls {
		if err := handleCall.flush(); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	s.handleCalls = nil
	return errs
}

// BufferedHandler is a slog.Handler that buffers log records in memory
// until Dispatch() is called. This allows for batching log operations
// which can be useful for performance or to ensure logs from related
// operations are grouped together.
//
// BufferedHandler wraps another handler that will process the log records
// when Dispatch() is called. All log records are stored in a shared buffer
// so that multiple instances created through WithAttrs() or WithGroup()
// will share the same underlying buffered logs.
//
// The buffer is not automatically flushed - you must call Dispatch() explicitly
// to process the buffered log records. If Dispatch() is not called, logs will
// remain in memory and not be processed by the underlying handler.
type BufferedHandler struct {
	handler      slog.Handler
	sharedBuffer *sharedBuffer
}

func NewBufferedHandler(handler slog.Handler) *BufferedHandler {
	return &BufferedHandler{
		handler:      handler,
		sharedBuffer: &sharedBuffer{},
	}
}

func (h *BufferedHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *BufferedHandler) Handle(ctx context.Context, record slog.Record) error {
	h.sharedBuffer.appendHandleCall(h.handler, ctx, record)
	return nil
}

func (h *BufferedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &BufferedHandler{
		handler:      h.handler.WithAttrs(attrs),
		sharedBuffer: h.sharedBuffer,
	}
}

func (h *BufferedHandler) WithGroup(name string) slog.Handler {
	return &BufferedHandler{
		handler:      h.handler.WithGroup(name),
		sharedBuffer: h.sharedBuffer,
	}
}

// Flush processes all buffered log records by sending them to the underlying handler.
// After a successful dispatch, the buffer is cleared.
// Returns an error if any of the buffered records fail to process.
//
// It is safe to call Flush() multiple times, though subsequent calls will have
// no effect until new log records are buffered.
func (h *BufferedHandler) Flush() error {
	return h.sharedBuffer.flush()
}
