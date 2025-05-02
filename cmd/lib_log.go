package main

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance"
	"github.com/fornellas/resonance/log"
)

var defaultLevel = slog.LevelInfo

type LogLevelValue slog.Level

func NewLogLevelValue() *LogLevelValue {
	logLevelValue := LogLevelValue(defaultLevel)
	return &logLevelValue
}

func (l LogLevelValue) String() string {
	return strings.ToLower(slog.Level(l).String())
}

func (l *LogLevelValue) Set(value string) error {
	return (*slog.Level)(l).UnmarshalText([]byte(value))
}

func (l *LogLevelValue) Reset() {
	if err := l.Set(defaultLevel.String()); err != nil {
		panic(err)
	}
}

func (l LogLevelValue) Type() string {
	return fmt.Sprintf("[%s]", strings.Join([]string{
		strings.ToLower(slog.LevelDebug.String()),
		strings.ToLower(slog.LevelInfo.String()),
		strings.ToLower(slog.LevelWarn.String()),
		strings.ToLower(slog.LevelError.String()),
	}, "|"))
}

func (l LogLevelValue) Level() slog.Level {
	return slog.Level(l)
}

var logLevelValue = NewLogLevelValue()

type LogHandlerValueOptions struct {
	Level             slog.Level
	AddSource         bool
	ConsoleTime       bool
	ConsoleForceColor bool
}

var logHandlerNameFnMap = map[string]func(io.Writer, LogHandlerValueOptions) slog.Handler{
	"console": func(writer io.Writer, options LogHandlerValueOptions) slog.Handler {
		var timeLayout string
		if options.ConsoleTime {
			timeLayout = time.DateTime
		}
		return log.NewConsoleHandler(writer, &log.ConsoleHandlerOptions{
			HandlerOptions: slog.HandlerOptions{
				Level:     options.Level,
				AddSource: options.AddSource,
			},
			TimeLayout: timeLayout,
			ForceColor: options.ConsoleForceColor,
		})
	},
	"json": func(writer io.Writer, options LogHandlerValueOptions) slog.Handler {
		return slog.NewJSONHandler(writer, &slog.HandlerOptions{
			AddSource: options.AddSource,
			Level:     options.Level,
		})
	},
}

func logHandlerNames() (names []string) {
	for name := range logHandlerNameFnMap {
		names = append(names, name)
	}
	return names
}

var defaultLogHandlerValue = "console"

type LogHandlerValue struct {
	name string
}

func NewLogHandlerValue() *LogHandlerValue {
	return &LogHandlerValue{name: defaultLogHandlerValue}
}

func (h *LogHandlerValue) String() string {
	return h.name
}

func (h *LogHandlerValue) Set(value string) error {
	if _, ok := logHandlerNameFnMap[value]; !ok {
		return fmt.Errorf("invalid log handler name '%s', valid options are %s", value, h.Type())
	}
	h.name = value
	return nil
}

func (h *LogHandlerValue) Reset() {
	if err := h.Set(defaultLogHandlerValue); err != nil {
		panic(err)
	}
}

func (h *LogHandlerValue) Type() string {
	return fmt.Sprintf("[%s]", strings.Join(logHandlerNames(), "|"))
}

func (h *LogHandlerValue) GetHandler(
	writer io.Writer, options LogHandlerValueOptions,
) slog.Handler {
	fn, ok := logHandlerNameFnMap[h.name]
	if !ok {
		panic("bug detected: invalid handler name")
	}
	return fn(writer, options)
}

var logHandlerValue = NewLogHandlerValue()

var defaultLogHandlerAddSource = false
var logHandlerAddSource = defaultLogHandlerAddSource

var defaultLogHandlerConsoleTime = false
var logHandlerConsoleTime = defaultLogHandlerConsoleTime

var defaultLogHandlerConsoleForceColor = false
var logHandlerConsoleForceColor = defaultLogHandlerConsoleForceColor

func AddLoggerFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().VarP(logLevelValue, "log-level", "l", "Logging level")

	cmd.PersistentFlags().VarP(logHandlerValue, "log-handler", "", "Logging handler")

	cmd.PersistentFlags().BoolVarP(
		&logHandlerAddSource, "log-handler-add-source", "", defaultLogHandlerAddSource,
		"Include source code position of the log statement when logging",
	)

	cmd.PersistentFlags().BoolVarP(
		&logHandlerConsoleTime, "log-handler-console-time", "", defaultLogHandlerConsoleTime,
		"Enable time for console handler",
	)

	cmd.PersistentFlags().BoolVarP(
		&logHandlerConsoleForceColor, "log-handler-console-force-color", "", defaultLogHandlerConsoleForceColor,
		"Force ANSI colors even when terminal is not detected",
	)
}

func GetLogger(writer io.Writer) *slog.Logger {
	handler := logHandlerValue.GetHandler(
		writer,
		LogHandlerValueOptions{
			Level:             logLevelValue.Level(),
			AddSource:         logHandlerAddSource,
			ConsoleTime:       logHandlerConsoleTime,
			ConsoleForceColor: logHandlerConsoleForceColor,
		},
	)
	return slog.New(handler).With("((o)) Resonance", resonance.Version)
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		logLevelValue.Reset()
		logHandlerValue.Reset()
		logHandlerAddSource = defaultLogHandlerAddSource
		logHandlerConsoleTime = defaultLogHandlerConsoleTime
		logHandlerConsoleForceColor = defaultLogHandlerConsoleForceColor
	})
}
