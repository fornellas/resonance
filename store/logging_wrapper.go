package store

import (
	"context"
	"io"
)

// Wraps a Store and log function calls.
type LoggingWrapper struct {
	store Store
}

func NewLoggingWrapper(store Store) *LoggingWrapper {
	return &LoggingWrapper{
		store: store,
	}
}

func (s *LoggingWrapper) GetLogWriterCloser(ctx context.Context, name string) (io.WriteCloser, error) {
	return s.store.GetLogWriterCloser(ctx, name)
}
