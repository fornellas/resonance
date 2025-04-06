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
	handler := NewConsoleHandler(os.Stderr, &ConsoleHandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
			ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
				if len(groups) == 0 && attr.Key == "version" {
					return slog.Attr{}
				}
				return attr
			},
		},
		Time: false,
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

// FIXME replace by slog.Handler.WithGroup
func MustContextLoggerWithSection(ctx context.Context, msg string, args ...any) (context.Context, *slog.Logger) {
	logger := MustLogger(ctx).WithGroup(msg).With(args...)
	return WithLogger(ctx, logger), logger
}
