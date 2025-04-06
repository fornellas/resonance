package main

import (
	"log/slog"
	"os"

	"github.com/fornellas/resonance/log"
)

func main() {
	// Create a ConsoleHandler
	handler := log.NewConsoleHandler(os.Stdout, &log.ConsoleHandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			// AddSource: true,
		},
		// Time: true,
	})

	// Create a logger
	logger := slog.New(handler)

	// Set as default
	slog.SetDefault(logger)

	// Root level logging
	slog.Info("Top message")

	// Group Foo
	fooLogger := logger.WithGroup("Foo").With(
		"foo", "foo",
	)

	fooLogger.Info("foo info")
	fooLogger.Error("foo error")

	barLogger := fooLogger.WithGroup("Bar").With(
		"bar", "bar",
	)

	barLogger.Info("bar info")
	fooLogger.Info("foo mixed")
	barLogger.Warn(
		"bar record attrs",
		"record multi line", "a\nb",
		"record single line", "line",
		slog.Group(
			"foo",
			"a", "b\nc",
		),
		slog.Group(
			"",
			"record empty", "group",
		),
	)
}
