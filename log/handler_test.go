package log

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func stripAnsiCodes(s string) string {
	r := strings.NewReplacer("\x1b[", "")
	return r.Replace(s)
}

func TestConsoleHandler(t *testing.T) {
	t.Run("Enabled", func(t *testing.T) {
		tests := []struct {
			name     string
			level    slog.Level
			minLevel slog.Level
			want     bool
		}{
			{
				name:     "debug level enabled when min level is debug",
				level:    slog.LevelDebug,
				minLevel: slog.LevelDebug,
				want:     true,
			},
			{
				name:     "info level enabled when min level is debug",
				level:    slog.LevelInfo,
				minLevel: slog.LevelDebug,
				want:     true,
			},
			{
				name:     "debug level not enabled when min level is info",
				level:    slog.LevelDebug,
				minLevel: slog.LevelInfo,
				want:     false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h := NewConsoleHandler(&bytes.Buffer{}, ConsoleHandlerOptions{
					Level: tt.minLevel,
				})
				got := h.Enabled(context.Background(), tt.level)
				if got != tt.want {
					t.Errorf("Enabled() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("Handle", func(t *testing.T) {
		tests := []struct {
			name    string
			record  slog.Record
			options ConsoleHandlerOptions
			want    string
		}{
			{
				name: "debug message",
				record: func() slog.Record {
					r := slog.NewRecord(time.Time{}, slog.LevelDebug, "debug message", 0)
					return r
				}(),
				options: ConsoleHandlerOptions{
					Level: slog.LevelDebug,
					Time:  false,
				},
				want: "debug message\n",
			},
			{
				name: "info message",
				record: func() slog.Record {
					r := slog.NewRecord(time.Time{}, slog.LevelInfo, "info message", 0)
					return r
				}(),
				options: ConsoleHandlerOptions{
					Level: slog.LevelDebug,
					Time:  false,
				},
				want: "info message\n",
			},
			{
				name: "warn message",
				record: func() slog.Record {
					r := slog.NewRecord(time.Time{}, slog.LevelWarn, "warn message", 0)
					return r
				}(),
				options: ConsoleHandlerOptions{
					Level: slog.LevelDebug,
					Time:  false,
				},
				want: "WARN warn message\n",
			},
			{
				name: "error message",
				record: func() slog.Record {
					r := slog.NewRecord(time.Time{}, slog.LevelError, "error message", 0)
					return r
				}(),
				options: ConsoleHandlerOptions{
					Level: slog.LevelDebug,
					Time:  false,
				},
				want: "ERROR error message\n",
			},
			{
				name: "with attributes",
				record: func() slog.Record {
					r := slog.NewRecord(time.Time{}, slog.LevelInfo, "message with attrs", 0)
					r.AddAttrs(slog.String("key", "value"))
					return r
				}(),
				options: ConsoleHandlerOptions{
					Level: slog.LevelDebug,
					Time:  false,
				},
				want: "message with attrs\n  key: value\n",
			},
			{
				name: "with time",
				record: func() slog.Record {
					r := slog.NewRecord(time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), slog.LevelInfo, "message with time", 0)
					return r
				}(),
				options: ConsoleHandlerOptions{
					Level: slog.LevelDebug,
					Time:  true,
				},
				want: "2023-01-01 12:00:00 message with time\n",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var buf bytes.Buffer
				h := NewConsoleHandler(&buf, tt.options)
				h.Handle(context.Background(), tt.record)

				got := stripAnsiCodes(buf.String())
				want := stripAnsiCodes(tt.want)

				if got != want {
					t.Errorf("Handle() got:\n%q\nwant:\n%q", got, want)
				}
			})
		}
	})

	t.Run("WithAttrs", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewConsoleHandler(&buf, ConsoleHandlerOptions{
			Level: slog.LevelDebug,
			Time:  false,
		})

		h2 := h.WithAttrs([]slog.Attr{
			slog.String("key1", "value1"),
			slog.Int("key2", 42),
		})

		record := slog.NewRecord(time.Time{}, slog.LevelInfo, "message", 0)
		record.AddAttrs(slog.String("key3", "value3"))

		h2.Handle(context.Background(), record)

		got := stripAnsiCodes(buf.String())
		want := stripAnsiCodes("message\n  key1: value1\n  key2: 42\n  key3: value3\n")

		if got != want {
			t.Errorf("WithAttrs() got:\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("WithGroup", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewConsoleHandler(&buf, ConsoleHandlerOptions{
			Level: slog.LevelDebug,
			Time:  false,
		})

		h2 := h.WithGroup("group1")
		h3 := h2.WithAttrs([]slog.Attr{slog.String("g1key", "g1value")})
		h4 := h3.WithGroup("group2")
		h5 := h4.WithAttrs([]slog.Attr{slog.String("g2key", "g2value")})

		record := slog.NewRecord(time.Time{}, slog.LevelInfo, "grouped message", 0)
		h5.Handle(context.Background(), record)

		got := stripAnsiCodes(buf.String())
		want := stripAnsiCodes("grouped message\n  group1:\n    g1key: g1value\n    group2:\n      g2key: g2value\n")

		if got != want {
			t.Errorf("WithGroup() got:\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("WithSection", func(t *testing.T) {
		var buf bytes.Buffer
		var h slog.Handler = NewConsoleHandler(&buf, ConsoleHandlerOptions{
			Level: slog.LevelDebug,
			Time:  false,
		})

		h2 := h.(SectionHandler).WithSection()

		record := slog.NewRecord(time.Time{}, slog.LevelInfo, "indented message", 0)
		h2.Handle(context.Background(), record)

		got := stripAnsiCodes(buf.String())
		want := stripAnsiCodes("  indented message\n")

		if got != want {
			t.Errorf("WithSection() got:\n%q\nwant:\n%q", got, want)
		}

		buf.Reset()
		h3 := h2.(SectionHandler).WithSection()
		record = slog.NewRecord(time.Time{}, slog.LevelInfo, "double indented", 0)
		h3.Handle(context.Background(), record)

		got = stripAnsiCodes(buf.String())
		want = stripAnsiCodes("    double indented\n")

		if got != want {
			t.Errorf("Nested WithSection() got:\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("ComplexScenario", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewConsoleHandler(&buf, ConsoleHandlerOptions{
			Level: slog.LevelDebug,
			Time:  false,
		})

		h2 := h.WithAttrs([]slog.Attr{slog.String("base", "value")})

		h3 := h2.WithGroup("config")

		h4 := h3.WithAttrs([]slog.Attr{
			slog.Int("timeout", 30),
			slog.Bool("enabled", true),
		})

		h5 := h4.(SectionHandler).WithSection()

		h6 := h5.WithGroup("request")

		h7 := h6.WithAttrs([]slog.Attr{
			slog.String("method", "GET"),
			slog.String("path", "/api/data"),
		})

		record := slog.NewRecord(time.Time{}, slog.LevelInfo, "processing request", 0)
		record.AddAttrs(slog.Int("status", 200))

		h7.Handle(context.Background(), record)

		got := stripAnsiCodes(buf.String())
		want := stripAnsiCodes("  processing request\n    base: value\n    config:\n      timeout: 30\n      enabled: true\n      request:\n        method: GET\n        path: /api/data\n        status: 200\n")

		if got != want {
			t.Errorf("Complex scenario got:\n%q\nwant:\n%q", got, want)
		}
	})
}
