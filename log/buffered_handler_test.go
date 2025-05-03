package log

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func log(logger *slog.Logger) {
	logger.Debug("debug msg", "debug", "attr")
	logger.Info("info msg", "info", "attr")
	logger.Warn("warn msg", "warn", "attr")
	logger.Error("error msg", "error", "attr")

	logger.With("attr", "logger").Info("info msg With")
	logger.WithGroup("group").With("with", "attr").Info("info msg WithGroup", "info", "attr")
	logger.With("attr", "again").Info("info attr", "info", "attr")
	logger.Info("info last")
}

func TestBufferedHandler(t *testing.T) {
	var buff bytes.Buffer

	handler := slog.NewTextHandler(&buff, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "time" {
				return slog.Attr{}
			}
			return a
		},
	})

	logger := slog.New(handler)
	log(logger)
	expectedWritten := buff.String()
	require.Greater(t, len(expectedWritten), 0)
	buff.Reset()

	bufferedHandler := NewBufferedHandler(handler)
	logger = slog.New(bufferedHandler)
	log(logger)
	require.Equal(t, buff.String(), "")
	require.NoError(t, bufferedHandler.Flush())
	require.Equal(t, expectedWritten, buff.String())
}
