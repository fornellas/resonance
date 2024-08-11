package main

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/internal"
	storePkg "github.com/fornellas/resonance/internal/store"
	"github.com/fornellas/resonance/log"
)

type LogLevelValue slog.Level

func NewLogLevelValue() *LogLevelValue {
	v := LogLevelValue(slog.LevelInfo)
	return &v
}

func (l LogLevelValue) String() string {
	return strings.ToLower(slog.Level(l).String())
}

func (l *LogLevelValue) Set(value string) error {
	return (*slog.Level)(l).UnmarshalText([]byte(value))
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
		return log.NewConsoleHandler(writer, log.ConsoleHandlerOptions{
			Level:     options.Level,
			AddSource: options.AddSource,
			Time:      options.ConsoleTime,
			ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
				if len(groups) == 0 && attr.Key == "version" {
					return slog.Attr{}
				}
				return attr
			},
		})
	},
	"json": func(writer io.Writer, options LogHandlerValueOptions) slog.Handler {
		return slog.NewJSONHandler(writer, &slog.HandlerOptions{
			AddSource: options.AddSource,
			Level:     options.Level,
			ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
				if len(groups) == 0 && attr.Key == slog.SourceKey {
					attr.Value = slog.AnyValue(
						strings.Replace(attr.Value.String(), " "+internal.GitTopLevel, " ", 1),
					)
				}
				return attr
			},
		})
	},
}

func logHandlerNames() (names []string) {
	for name := range logHandlerNameFnMap {
		names = append(names, name)
	}
	return names
}

type LogHandlerValue struct {
	name string
}

func NewLogHandlerValue() *LogHandlerValue {
	return &LogHandlerValue{name: "console"}
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

var storeNameMap = map[string]bool{
	"target":    true,
	"localhost": true,
}

type StoreValue struct {
	name string
}

func NewStoreValue() *StoreValue {
	return &StoreValue{
		name: "target",
	}
}

func (s *StoreValue) String() string {
	return s.name
}

func (s *StoreValue) Set(value string) error {
	if _, ok := storeNameMap[value]; !ok {
		return fmt.Errorf("invalid store name '%s', valid options are %s", value, s.Type())
	}
	s.name = value
	return nil
}

func (s *StoreValue) Type() string {
	names := []string{}
	for name := range storeNameMap {
		names = append(names, name)
	}
	return fmt.Sprintf("[%s]", strings.Join(names, "|"))
}

func (s *StoreValue) GetStore(hst host.Host, path string) storePkg.Store {
	return storePkg.NewHostStore(hst, path)
}
