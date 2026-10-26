package store

import (
	"context"
	"io"

	"github.com/fornellas/resonance/state"
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

func (s *LoggingWrapper) GetOriginalState(ctx context.Context) (*state.State, error) {
	return s.store.GetOriginalState(ctx)
}

func (s *LoggingWrapper) SaveOriginalState(ctx context.Context, state *state.State) error {
	return s.store.SaveOriginalState(ctx, state)
}

func (s *LoggingWrapper) GetCommittedState(ctx context.Context) (*state.State, error) {
	return s.store.GetCommittedState(ctx)
}

func (s *LoggingWrapper) CommitPlannedState(ctx context.Context) error {
	return s.store.CommitPlannedState(ctx)
}

func (s *LoggingWrapper) GetPlannedState(ctx context.Context) (*state.State, error) {
	return s.store.GetPlannedState(ctx)
}

func (s *LoggingWrapper) SavePlannedState(ctx context.Context, state *state.State) error {
	return s.store.SavePlannedState(ctx, state)
}

func (s *LoggingWrapper) GetLogWriterCloser(ctx context.Context, name string) (io.WriteCloser, error) {
	return s.store.GetLogWriterCloser(ctx, name)
}
