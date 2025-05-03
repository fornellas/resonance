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
	handler := NewTerminalTreeHandler(os.Stderr, &TerminalHandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
			ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
				if len(groups) == 0 && attr.Key == "((o)) Resonance" {
					return slog.Attr{}
				}
				return attr
			},
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

// Returns a copy of the given context with a logger that has a group added to it.
// The logger is retrieved from the context with [MustLogger] and a group is added to it with
// [slog.Logger.MustWithGroup], then the new logger is stored in the returned context.
func MustWithGroup(ctx context.Context, name string) (context.Context, *slog.Logger) {
	logger := MustLogger(ctx).WithGroup(name)
	return WithLogger(ctx, logger), logger
}

// Returns a copy of the given context with a logger that has attributes added to it.
// The logger is retrieved from the context with [MustLogger] and attributes are added to it with
// [slog.Logger.With], then the new logger is stored in the returned context.
func MustWithAttrs(ctx context.Context, args ...any) (context.Context, *slog.Logger) {
	logger := MustLogger(ctx).With(args...)
	return WithLogger(ctx, logger), logger
}

// Returns a copy of the given context with a logger that has a group and attributes added to it.
// The logger is retrieved from the context with [MustLogger], a group is added to it with
// [slog.Logger.WithGroup], and attributes are added with [slog.Logger.With],
// then the new logger is stored in the returned context.
func MustWithGroupAttrs(ctx context.Context, name string, args ...any) (context.Context, *slog.Logger) {
	logger := MustLogger(ctx).WithGroup(name).With(args...)
	return WithLogger(ctx, logger), logger
}
