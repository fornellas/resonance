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

// MustContextLoggerSection creates a new log section.
// It fetches a logger value from the context, previously set with WithLogger, and logs given
// msg and args as info.
// If its handler implements [SectionHandler], then it WithSection(), and set the new logger value
// to a copy of the context.
// If the handler does not implement [SectionHandler], return the original context and the
// retrieved logger.
// It panics if no logger value has been set previously.
func MustContextLoggerWithSection(ctx context.Context, msg string, args ...any) (context.Context, *slog.Logger) {
	logger := MustLogger(ctx)
	logger.Info(msg, args...)
	if logger.Handler().Enabled(ctx, slog.LevelInfo) {
		handler, ok := logger.Handler().(SectionHandler)
		if ok {
			logger = slog.New(handler.WithSection())
			ctx = WithLogger(ctx, logger)
		}
	}
	return ctx, logger
}
