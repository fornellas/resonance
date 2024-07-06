package log

import (
	"context"
	"log/slog"
	"os"
)

type loggerKeyType struct{}

var loggerKey loggerKeyType

// Returns a copy of the given context with the logger value set. The value can be retreived
// with [MustLogger], [MustContextLoggerIndented] or [MustLoggerIndented].
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// Similar to [WithLogger], but constructs a new logger suitable for using during testss.
func WithTestLogger(ctx context.Context) context.Context {
	handler := NewConsoleHandler(os.Stderr, ConsoleHandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		Time:      false,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if len(groups) == 0 && attr.Key == "version" {
				return slog.Attr{}
			}
			return attr
		},
	})
	logger := slog.New(handler)
	return WithLogger(ctx, logger)
}

// Returns the logger value associated with the context, which must have been previously set with
// [WithLogger].
// It panics if no logger value has been set previously.
func MustLogger(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerKey).(*slog.Logger)
	if !ok {
		panic("bug detected: context has no logger set")
	}
	return logger
}

// Retrieves the logger value from the context, previously set with [WithLogger]. If its handler
// implements [IndentableHandler], then it WithIndent(), and set the new logger value to a
// copy of the context. If the handler does not implement [IndentableHandler], return the original
// context and the retrieved logger.
// It panics if no logger value has been set previously.
func MustContextLoggerIndented(ctx context.Context) (context.Context, *slog.Logger) {
	logger := MustLogger(ctx)
	handler, ok := logger.Handler().(IndentableHandler)
	if ok {
		logger = slog.New(handler.WithIndent())
		ctx = WithLogger(ctx, logger)
	}
	return ctx, logger
}

// Similar to [MustContextLoggerIndented], but only returns the logger, not the context.
func MustLoggerIndented(ctx context.Context) *slog.Logger {
	_, logger := MustContextLoggerIndented(ctx)
	return logger
}
