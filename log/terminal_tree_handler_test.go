package log

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminalTreeHandler(t *testing.T) {
	t.Run("Interface", func(t *testing.T) {
		var _ slog.Handler = &TerminalTreeHandler{}
	})

	t.Run("Enabled", func(t *testing.T) {
		tests := []struct {
			name     string
			level    slog.Level
			minLevel slog.Level
			want     bool
		}{
			{"debug-info", slog.LevelDebug, slog.LevelInfo, false},
			{"info-info", slog.LevelInfo, slog.LevelInfo, true},
			{"warn-info", slog.LevelWarn, slog.LevelInfo, true},
			{"error-info", slog.LevelError, slog.LevelInfo, true},
			{"debug-debug", slog.LevelDebug, slog.LevelDebug, true},
			{"info-debug", slog.LevelInfo, slog.LevelDebug, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var levelVar slog.LevelVar
				levelVar.Set(tt.minLevel)

				h := NewTerminalTreeHandler(&bytes.Buffer{}, &TerminalHandlerOptions{
					HandlerOptions: slog.HandlerOptions{
						Level: &levelVar,
					},
				})

				got := h.Enabled(context.Background(), tt.level)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("Handle", func(t *testing.T) {
		tests := []struct {
			name        string
			setupLogger func(buf *bytes.Buffer) *slog.Logger
			logFunc     func(*slog.Logger)
			check       func(t *testing.T, output string)
		}{
			{
				name: "simple_message",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{NoColor: true})
					return slog.New(h)
				},
				logFunc: func(logger *slog.Logger) {
					logger.Info("hello world")
				},
				check: func(t *testing.T, output string) {
					assert.Equal(t, "INFO hello world\n", output)
				},
			},
			{
				name: "with_attrs",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{NoColor: true})
					return slog.New(h.WithAttrs([]slog.Attr{
						slog.String("service", "test"),
						slog.Int("version", 1),
					}))
				},
				logFunc: func(logger *slog.Logger) {
					logger.Info("hello with attrs", "user", "tester")
					logger = logger.With("extra", "attr")
					logger.Info("hello with extra attrs", "something", "else")
				},
				check: func(t *testing.T, output string) {
					assert.Equal(
						t,
						"service: test\n"+
							"version: 1\n"+
							"INFO hello with attrs\n"+
							"  user: tester\n"+
							"extra: attr\n"+
							"INFO hello with extra attrs\n"+
							"  something: else\n",
						output,
					)
				},
			},
			{
				name: "with_groups",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{NoColor: true})
					return slog.New(h.WithGroup("server").WithAttrs([]slog.Attr{
						slog.String("type", "api"),
					}))
				},
				logFunc: func(logger *slog.Logger) {
					logger.Info("started server", "port", 8080)
					logger = logger.With("extra", "attr")
					logger.Info("with extra attrs", "something", "else")
				},
				check: func(t *testing.T, output string) {
					assert.Equal(
						t,
						"🏷️ server\n"+
							"  type: api\n"+
							"  INFO started server\n"+
							"    port: 8080\n"+
							"  extra: attr\n"+
							"  INFO with extra attrs\n"+
							"    something: else\n",
						output,
					)
				},
			},
			{
				name: "with_similar_groups",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{NoColor: true})
					return slog.New(h)
				},
				logFunc: func(logger *slog.Logger) {
					logger1 := logger.WithGroup("Same Group")
					logger1.Info("first")
					logger2 := logger.WithGroup("Same Group")
					logger2.Info("second")
				},
				check: func(t *testing.T, output string) {
					assert.Equal(
						t,
						"🏷️ Same Group\n"+
							"  INFO first\n"+
							"  INFO second\n",
						output,
					)
				},
			},
			{
				// Test for the case where we switch between handlers with different groups
				// This tests the code path marked with "FIXME add test" in writeHandlerGroupAttrs
				name: "different_groups",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{NoColor: true})
					// Just return the base handler, as we'll create multiple loggers in logFunc
					return slog.New(h)
				},
				logFunc: func(logger *slog.Logger) {
					// Get the base handler from the logger
					h := logger.Handler().(*TerminalTreeHandler)

					// First log with one group
					logger1 := slog.New(h.WithGroup("group1").WithAttrs([]slog.Attr{
						slog.String("attr1", "value1"),
					}))
					logger1.Info("first message")

					// Then log with a different group
					logger2 := slog.New(h.WithGroup("group2").WithAttrs([]slog.Attr{
						slog.String("attr2", "value2"),
					}))
					logger2.Info("second message")
				},
				check: func(t *testing.T, output string) {
					expected := "🏷️ group1\n" +
						"  attr1: value1\n" +
						"  INFO first message\n" +
						"🏷️ group2\n" +
						"  attr2: value2\n" +
						"  INFO second message\n"
					assert.Equal(t, expected, output)
				},
			},
			{
				name: "different_groups",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{NoColor: true})
					return slog.New(h)
				},
				logFunc: func(logger *slog.Logger) {
					h := logger.Handler().(*TerminalTreeHandler)

					logger1 := slog.New(h.WithGroup("group1").WithAttrs([]slog.Attr{
						slog.String("attr1", "value1"),
					}))
					logger1.Info("first message")

					logger2 := slog.New(h.WithGroup("group2").WithAttrs([]slog.Attr{
						slog.String("attr2", "value2"),
					}))
					logger2.Info("second message")
				},
				check: func(t *testing.T, output string) {
					expected := "🏷️ group1\n" +
						"  attr1: value1\n" +
						"  INFO first message\n" +
						"🏷️ group2\n" +
						"  attr2: value2\n" +
						"  INFO second message\n"
					assert.Equal(t, expected, output)
				},
			},
			{
				name: "nested_groups",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{NoColor: true})
					return slog.New(
						h.WithGroup("app").
							WithAttrs([]slog.Attr{slog.String("version", "1.0")}).
							WithGroup("database").
							WithAttrs([]slog.Attr{slog.String("db", "sql")}),
					)
				},
				logFunc: func(logger *slog.Logger) {
					logger.Info("connected", "host", "localhost", "port", 5432)
				},
				check: func(t *testing.T, output string) {
					assert.Equal(
						t,
						"🏷️ app\n"+
							"  version: 1.0\n"+
							"  🏷️ database\n"+
							"    db: sql\n"+
							"    INFO connected\n"+
							"      host: localhost\n"+
							"      port: 5432\n",
						output,
					)
				},
			},
			{
				name: "group_in_record",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{NoColor: true})
					return slog.New(h)
				},
				logFunc: func(logger *slog.Logger) {
					logger.Info("user action",
						slog.Group("user",
							slog.String("id", "123"),
							slog.String("name", "test"),
						),
						"action", "login",
					)
				},
				check: func(t *testing.T, output string) {
					assert.Equal(
						t,
						"INFO user action\n"+
							"  🏷️ user\n"+
							"    id: 123\n"+
							"    name: test\n"+
							"  action: login\n",
						output,
					)
				},
			},
			{
				name: "multiline_string",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{NoColor: true})
					return slog.New(h)
				},
				logFunc: func(logger *slog.Logger) {
					logger.Error("error occurred", "stack", "line1\nline2\nline3\t tab")
				},
				check: func(t *testing.T, output string) {
					assert.Equal(
						t,
						"ERROR error occurred\n"+
							"  stack:\n"+
							"    line1\n"+
							"    line2\n"+
							"    line3\t tab\n",
						output,
					)
				},
			},
			{
				name: "with_timestamp",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{
						NoColor:    true,
						TimeLayout: time.RFC3339,
					})
					return slog.New(h)
				},
				logFunc: func(logger *slog.Logger) {
					logger.Info("message with time")
				},
				check: func(t *testing.T, output string) {
					assert.Regexp(t, `^INFO message with time
  ((?:(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2}:\d{2}(?:\.\d+)?))(Z|[\+-]\d{2}:\d{2})?)
$`, output)
				},
			},
			{
				name: "with_source",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{
						NoColor: true,
						HandlerOptions: slog.HandlerOptions{
							AddSource: true,
						},
					})
					return slog.New(h)
				},
				logFunc: func(logger *slog.Logger) {
					logger.Info("message with source")
				},
				check: func(t *testing.T, output string) {
					assert.Regexp(t, `^INFO message with source\n  .+terminal_tree_handler_test\.go:\d+ \(.+\)\n$`, output)
				},
			},
			{
				name: "empty_group",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{NoColor: true})
					return slog.New(h.WithGroup(""))
				},
				logFunc: func(logger *slog.Logger) {
					logger.Info("empty group should be ignored")
				},
				check: func(t *testing.T, output string) {
					assert.Equal(
						t,
						"INFO empty group should be ignored\n",
						output,
					)
				},
			},
			{
				name: "replace_attr",
				setupLogger: func(buf *bytes.Buffer) *slog.Logger {
					h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{
						NoColor: true,
						HandlerOptions: slog.HandlerOptions{
							ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
								if a.Key == "sensitive" {
									return slog.String("sensitive", "REDACTED")
								}
								return a
							},
						},
					})
					return slog.New(h)
				},
				logFunc: func(logger *slog.Logger) {
					logger.Info("message with sensitive data", "sensitive", "secret123", "normal", "visible")
				},
				check: func(t *testing.T, output string) {
					assert.Equal(
						t,
						"INFO message with sensitive data\n"+
							"  sensitive: REDACTED\n"+
							"  normal: visible\n",
						output,
					)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				buf := &bytes.Buffer{}
				logger := tt.setupLogger(buf)
				tt.logFunc(logger)
				tt.check(t, buf.String())
			})
		}
	})

	t.Run("WithGroup", func(t *testing.T) {
		buf := &bytes.Buffer{}
		h := NewTerminalTreeHandler(buf, &TerminalHandlerOptions{NoColor: true})

		// Empty group name should return same handler
		h2 := h.WithGroup("")
		require.Same(t, h, h2)

		// Non-empty group should return new handler
		h3 := h.WithGroup("test")
		require.NotSame(t, h, h3)

		logger := slog.New(h3)
		logger.Info("grouped message")

		output := buf.String()
		assert.Equal(
			t,
			"🏷️ test\n"+
				"  INFO grouped message\n",
			output,
		)
	})

	t.Run("ColorDetection", func(t *testing.T) {
		tests := []struct {
			name       string
			opts       *TerminalHandlerOptions
			wantColors bool
		}{
			{
				name:       "default_no_tty",
				opts:       nil,
				wantColors: false,
			},
			{
				name:       "force_color",
				opts:       &TerminalHandlerOptions{ForceColor: true},
				wantColors: true,
			},
			{
				name:       "no_color_takes_precedence",
				opts:       &TerminalHandlerOptions{ForceColor: true, NoColor: true},
				wantColors: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				buf := &bytes.Buffer{} // buffer is not a TTY
				h := NewTerminalTreeHandler(buf, tt.opts)

				logger := slog.New(h)
				logger.Error("test message")

				output := buf.String()
				hasColorCodes := strings.Contains(output, "\033[")
				assert.Equal(t, tt.wantColors, hasColorCodes)
			})
		}
	})
}
