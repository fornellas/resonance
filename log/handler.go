package log

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"

	"github.com/fornellas/resonance/internal"
)

// This interface extends slog.Handler to enable logging with indentation.
type IndentableHandler interface {
	slog.Handler
	// Returns a copy of the handler with another level of indentation added.
	WithIndent() slog.Handler
}

// Options for [ConsoleHandler].
type ConsoleHandlerOptions struct {
	Level slog.Leveler
	// Whether to log the source file / line / module / function position of the log statement.
	AddSource bool
	// Whether to log the time.
	Time bool
	// ReplaceAttr is called to rewrite each non-group attribute before it is logged.
	ReplaceAttr func(groups []string, attr slog.Attr) slog.Attr
}

// Colored logging to the console, with indentation via [IndentableHandler] interface.
type ConsoleHandler struct {
	writer          io.Writer
	writerMutex     *sync.Mutex
	options         ConsoleHandlerOptions
	indentLevel     int
	attrIndentLevel int
	groups          []string
	groupLine       string
	groupAttrLines  []string
}

func NewConsoleHandler(writer io.Writer, options ConsoleHandlerOptions) *ConsoleHandler {
	h := &ConsoleHandler{
		writer:          writer,
		writerMutex:     &sync.Mutex{},
		options:         options,
		attrIndentLevel: 1,
	}
	return h
}

func (h *ConsoleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.options.Level.Level()
}

func (h *ConsoleHandler) escape(s string) string {
	rs := []rune{}
	for _, r := range s {
		if strconv.IsPrint(r) {
			rs = append(rs, r)
		} else {
			e := strconv.QuoteRune(r)
			e = e[1 : len(e)-1]
			rs = append(rs, []rune(e)...)
		}
	}
	return string(rs)
}

func (h *ConsoleHandler) getAttrLines(attr slog.Attr) []string {
	keyColor := color.New(color.FgCyan, color.Faint)
	valueColor := color.New(color.FgWhite, color.Faint)

	value := strings.TrimSuffix(attr.Value.Resolve().String(), "\n")
	valueLines := strings.Split(value, "\n")
	value = h.escape(value)
	for i, line := range valueLines {
		valueLines[i] = h.escape(line)
	}

	if len(valueLines) == 1 {
		var buff bytes.Buffer
		keyColor.Fprintf(&buff, "%*s%s: ", h.attrIndentLevel*2, "", attr.Key)
		valueColor.Fprintf(&buff, "%s", value)
		return []string{buff.String()}
	} else {
		lines := []string{}
		lines = append(lines, keyColor.Sprintf("%*s%s:", h.attrIndentLevel*2, "", attr.Key))

		for _, line := range valueLines {
			lines = append(lines, valueColor.Sprintf("%*s%s", (h.attrIndentLevel+1)*2, "", line))
		}

		return lines
	}
}

func (h *ConsoleHandler) Handle(ctx context.Context, record slog.Record) error {
	var lines []string
	var line string

	// Message
	line = ""
	switch record.Level {
	case slog.LevelDebug:
		line += color.New(color.FgWhite).Sprintf("%s", record.Message)
	case slog.LevelInfo:
		line += color.New(color.FgWhite, color.Bold).Sprintf("%s", record.Message)
	case slog.LevelWarn:
		line += color.New(color.FgYellow).Sprintf("%s ", record.Level.String())
		line += color.New(color.FgWhite, color.Bold).Sprintf("%s", record.Message)
	case slog.LevelError:
		line += color.New(color.FgRed).Sprintf("%s ", record.Level.String())
		line += color.New(color.FgWhite, color.Bold).Sprintf("%s", record.Message)
	default:
		panic("bug detected: invalid level")
	}
	lines = append(lines, line)

	// PC
	if record.PC != 0 && h.options.AddSource {
		line = ""
		frame, _ := runtime.CallersFrames([]uintptr{record.PC}).Next()
		color := color.New(color.FgMagenta, color.Faint)
		line += color.Sprintf("  %s:%d", strings.TrimPrefix(frame.File, internal.GitTopLevel), frame.Line)
		if len(frame.Function) > 0 {
			line += color.Sprintf(" (%s)", frame.Function)
		}
		lines = append(lines, line)
	}

	// Attrs
	var recordAttrLines []string
	record.Attrs(func(attr slog.Attr) bool {
		if h.options.ReplaceAttr != nil {
			attr = h.options.ReplaceAttr(h.groups, attr)
		}

		if attr.Equal(slog.Attr{}) {
			return false
		}

		recordAttrLines = append(recordAttrLines, h.getAttrLines(attr)...)

		return true
	})

	// Time
	var timePrefix string
	var timePrefixIndent string
	if !record.Time.IsZero() && h.options.Time {
		rawTimePrefix := record.Time.Format(time.DateTime) + " "
		timePrefix = color.New(color.FgWhite, color.Faint).Sprintf("%s", rawTimePrefix)
		timePrefixIndent = strings.Repeat(" ", len(rawTimePrefix))
	}

	// Buffer
	var buff bytes.Buffer
	color.New(color.Reset).Fprintf(&buff, "")
	for i, line := range append(lines, append(h.groupAttrLines, recordAttrLines...)...) {
		var prefix string
		if i == 0 {
			prefix = timePrefix
		} else {
			prefix = timePrefixIndent
		}
		if _, err := fmt.Fprintf(&buff, "%s%*s%s\n", prefix, h.indentLevel*2, "", line); err != nil {
			return err
		}
	}

	// Write
	h.writerMutex.Lock()
	defer h.writerMutex.Unlock()
	_, err := buff.WriteTo(h.writer)
	return err
}

func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h

	copy(h2.groups, h.groups)

	copy(h2.groupAttrLines, h.groupAttrLines)

	for _, attr := range attrs {
		if h.options.ReplaceAttr != nil {
			attr = h.options.ReplaceAttr(h.groups, attr)
		}

		if attr.Equal(slog.Attr{}) {
			continue
		}

		if len(h2.groupLine) > 0 {
			h2.groupAttrLines = append(h2.groupAttrLines, h2.groupLine)
			h2.groupLine = ""
			h.groupLine = ""
		}

		h2.groupAttrLines = append(h2.groupAttrLines, h2.getAttrLines(attr)...)
	}

	return &h2
}

func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	h2 := *h

	h2.attrIndentLevel += 1

	h2.groups = make([]string, len(h.groups)+1)
	copy(h2.groups, h.groups)
	h2.groups[len(h2.groups)-1] = name

	h2.groupLine = color.New(color.FgCyan, color.Faint).Sprintf("%*s%s:", h.attrIndentLevel*2, "", name)

	copy(h2.groupAttrLines, h.groupAttrLines)

	return &h2
}

func (h *ConsoleHandler) WithIndent() slog.Handler {
	h2 := *h

	h2.indentLevel += 1

	copy(h2.groups, h.groups)

	copy(h2.groupAttrLines, h.groupAttrLines)

	return &h2
}
