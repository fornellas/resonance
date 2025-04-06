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

func TestConsoleHandler_Compliance(t *testing.T) {
	// Test that ConsoleHandler implements slog.Handler
	var _ slog.Handler = &ConsoleHandler{}
}

func TestConsoleHandler_Enabled(t *testing.T) {
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

			h := NewConsoleHandler(&bytes.Buffer{}, &ConsoleHandlerOptions{
				HandlerOptions: slog.HandlerOptions{
					Level: &levelVar,
				},
			})

			got := h.Enabled(context.Background(), tt.level)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConsoleHandler_Handle(t *testing.T) {
	tests := []struct {
		name        string
		setupLogger func(buf *bytes.Buffer) *slog.Logger
		logFunc     func(*slog.Logger)
		check       func(t *testing.T, output string)
	}{
		{
			name: "simple_message",
			setupLogger: func(buf *bytes.Buffer) *slog.Logger {
				h := NewConsoleHandler(buf, &ConsoleHandlerOptions{NoColor: true})
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
				h := NewConsoleHandler(buf, &ConsoleHandlerOptions{NoColor: true})
				return slog.New(h.WithAttrs([]slog.Attr{
					slog.String("service", "test"),
					slog.Int("version", 1),
				}))
			},
			logFunc: func(logger *slog.Logger) {
				logger.Info("hello with attrs", "user", "tester")
			},
			check: func(t *testing.T, output string) {
				assert.Equal(
					t,
					"service: test\n"+
						"version: 1\n"+
						"INFO hello with attrs\n"+
						"  user: tester\n",
					output,
				)
			},
		},
		{
			name: "with_groups",
			setupLogger: func(buf *bytes.Buffer) *slog.Logger {
				h := NewConsoleHandler(buf, &ConsoleHandlerOptions{NoColor: true})
				return slog.New(h.WithGroup("server").WithAttrs([]slog.Attr{
					slog.String("type", "api"),
				}))
			},
			logFunc: func(logger *slog.Logger) {
				logger.Info("started server", "port", 8080)
			},
			check: func(t *testing.T, output string) {
				assert.Equal(
					t,
					"Group: server\n"+
						"  type: api\n"+
						"  INFO started server\n"+
						"    port: 8080\n",
					output,
				)
			},
		},
		{
			name: "nested_groups",
			setupLogger: func(buf *bytes.Buffer) *slog.Logger {
				h := NewConsoleHandler(buf, &ConsoleHandlerOptions{NoColor: true})
				return slog.New(h.WithGroup("app").WithGroup("database"))
			},
			logFunc: func(logger *slog.Logger) {
				logger.Info("connected", "host", "localhost", "port", 5432)
			},
			check: func(t *testing.T, output string) {
				assert.Equal(
					t,
					"Group: app\n"+
						"  Group: database\n"+
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
				h := NewConsoleHandler(buf, &ConsoleHandlerOptions{NoColor: true})
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
						"  Group: user\n"+
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
				h := NewConsoleHandler(buf, &ConsoleHandlerOptions{NoColor: true})
				return slog.New(h)
			},
			logFunc: func(logger *slog.Logger) {
				logger.Error("error occurred", "stack", "line1\nline2\nline3")
			},
			check: func(t *testing.T, output string) {
				assert.Equal(
					t,
					"ERROR error occurred\n"+
						"  stack:\n"+
						"    line1\n"+
						"    line2\n"+
						"    line3\n",
					output,
				)
			},
		},
		{
			name: "with_timestamp",
			setupLogger: func(buf *bytes.Buffer) *slog.Logger {
				h := NewConsoleHandler(buf, &ConsoleHandlerOptions{
					NoColor:    true,
					TimeLayout: time.RFC3339,
				})
				return slog.New(h)
			},
			logFunc: func(logger *slog.Logger) {
				logger.Info("message with time")
			},
			check: func(t *testing.T, output string) {
				// Check if output contains an RFC3339 date format (yyyy-mm-ddThh:mm:ss)
				assert.Regexp(t, `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[-+:0-9]* INFO message with time\n$`, output)
			},
		},
		{
			name: "with_source",
			setupLogger: func(buf *bytes.Buffer) *slog.Logger {
				h := NewConsoleHandler(buf, &ConsoleHandlerOptions{
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
				assert.Regexp(t, `^INFO message with source\n  .+console_handler_test\.go:\d+ \(.+\)\n$`, output)
			},
		},
		{
			name: "empty_group",
			setupLogger: func(buf *bytes.Buffer) *slog.Logger {
				h := NewConsoleHandler(buf, &ConsoleHandlerOptions{NoColor: true})
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
				h := NewConsoleHandler(buf, &ConsoleHandlerOptions{
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
}

func TestConsoleHandler_WithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	h := NewConsoleHandler(buf, &ConsoleHandlerOptions{NoColor: true})

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
		"Group: test\n"+
			"  INFO grouped message\n",
		output,
	)
}

func TestConsoleHandler_MultipleHandlers(t *testing.T) {
	buf := &bytes.Buffer{}
	h1 := NewConsoleHandler(buf, &ConsoleHandlerOptions{NoColor: true})
	h2 := h1.WithGroup("group1")

	logger1 := slog.New(h1)
	logger2 := slog.New(h2)

	logger1.Info("from logger1")
	output := buf.String()
	assert.Equal(t, "INFO from logger1\n", output)

	buf.Reset()

	logger2.Info("from logger2")
	output = buf.String()
	assert.Equal(t, "Group: group1\n  INFO from logger2\n", output)
}

func TestConsoleHandler_ColorDetection(t *testing.T) {
	tests := []struct {
		name       string
		opts       *ConsoleHandlerOptions
		wantColors bool
	}{
		{
			name:       "default_no_tty",
			opts:       nil,
			wantColors: false,
		},
		{
			name:       "force_color",
			opts:       &ConsoleHandlerOptions{ForceColor: true},
			wantColors: true,
		},
		{
			name:       "no_color_takes_precedence",
			opts:       &ConsoleHandlerOptions{ForceColor: true, NoColor: true},
			wantColors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{} // buffer is not a TTY
			h := NewConsoleHandler(buf, tt.opts)

			// Check if color is enabled
			assert.Equal(t, tt.wantColors, h.color)

			// Log something and check for color codes
			logger := slog.New(h)
			logger.Error("test message")

			output := buf.String()
			hasColorCodes := strings.Contains(output, "\033[")
			assert.Equal(t, tt.wantColors, hasColorCodes)
		})
	}
}
