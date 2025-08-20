package main

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/resonance"
)

var ApplyCmd = &cobra.Command{
	Use:   "apply [flags] [file|dir]",
	Short: "Apply resources.",
	Long:  "Load resources from file/dir and apply them.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		ctx, logger := log.MustWithGroupAttrs(cmd.Context(), "✏️ Apply", "path", path)

		var retErr error
		defer func() {
			if retErr != nil {
				logger.Error("Failed", "err", retErr)
				Exit(1)
			}
		}()

		host, ctx, err := GetHost(ctx)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to get host: %w", err))
			return
		}
		defer func() {
			if err := host.Close(ctx); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to close host: %w", err))
			}
		}()
		ctx, _ = log.MustWithAttrs(ctx, "host", fmt.Sprintf("%s => %s", host.Type(), host.String()))

		store, storeConfig, err := GetStore(host)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to get store: %w", err))
			return
		}
		ctx, logger = log.MustWithAttrs(ctx, "store", fmt.Sprintf("%s %s", storeValue.String(), storeConfig))

		storelogWriterCloser, err := store.GetLogWriterCloser(ctx, "apply")
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to get store log writer: %w", err))
			return
		}
		defer func() {
			if err := storelogWriterCloser.Close(); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to close store log: %w", err))
			}
		}()

		logHandler := logger.Handler()
		storeHandler := log.NewTerminalLineHandler(storelogWriterCloser, &log.TerminalHandlerOptions{
			HandlerOptions: slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug,
			},
			TimeLayout: time.RFC3339,
			NoColor:    true,
		}).
			WithAttrs([]slog.Attr{slog.String("version", resonance.Version)}).
			WithGroup("✏️ Apply").WithAttrs([]slog.Attr{slog.String("path", path)}).
			WithAttrs([]slog.Attr{slog.String("host", fmt.Sprintf("%s => %s", host.Type(), host.String()))}).
			WithAttrs([]slog.Attr{slog.String("store", fmt.Sprintf("%s %s", storeValue.String(), storeConfig))})
		logger = slog.New(log.NewMultiHandler(logHandler, storeHandler))
		ctx = log.WithLogger(ctx, logger)
		cmd.SetContext(ctx)

		panic("TODO")
	},
}

func init() {
	AddHostFlags(ApplyCmd)

	AddStoreFlags(ApplyCmd)

	RootCmd.AddCommand(ApplyCmd)
}
