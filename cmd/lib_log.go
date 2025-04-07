package main

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

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

type LogHandlerValueOptions struct {
	Level       slog.Level
	AddSource   bool
	ConsoleTime bool
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
