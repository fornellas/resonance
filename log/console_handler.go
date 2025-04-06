package log

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// ConsoleHandlerOptions extends HandlerOptions with console-specific options
type ConsoleHandlerOptions struct {
	slog.HandlerOptions
	// If true, include time in output
	Time bool
	// TODO Force ANSI escape sequences for color, even when no TTY detected.
	// ForceColor bool
	// TODO Disable color, even when TTY detected
	// NoColor bool
}

// ConsoleHandler implements slog.Handler
type ConsoleHandler struct {
	opts   *ConsoleHandlerOptions
	writer io.Writer
	isTTY  bool
	groups []string
	attrs  []slog.Attr
}

// NewConsoleHandler creates a new ConsoleHandler
func NewConsoleHandler(w io.Writer, opts *ConsoleHandlerOptions) *ConsoleHandler {
	if opts == nil {
		opts = &ConsoleHandlerOptions{}
	}

	isTTY := false
	if f, ok := w.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	return &ConsoleHandler{
		opts:   opts,
		writer: w,
		isTTY:  isTTY,
		groups: []string{},
	}
}

func (h *ConsoleHandler) clone() *ConsoleHandler {
	h2 := *h
	h2.groups = slices.Clip(h.groups)
	h2.attrs = slices.Clip(h.attrs)
	return &h2
}

// Enabled implements slog.Handler.Enabled
func (h *ConsoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

// WithAttrs implements slog.Handler.WithAttrs
func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := h.clone()
	h2.attrs = append(h2.attrs, attrs...)
	return h2
}

// WithGroup implements slog.Handler.WithGroup
func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	if len(name) == 0 {
		return h
	}
	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
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

func (h *ConsoleHandler) writeAttr(writer io.Writer, indent int, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if h.opts.ReplaceAttr != nil && attr.Value.Kind() != slog.KindGroup {
		attr = h.opts.ReplaceAttr(h.groups, attr)
		attr.Value = attr.Value.Resolve()
	}

	if attr.Equal(slog.Attr{}) {
		return
	}

	indentStr := strings.Repeat("  ", indent)

	if attr.Value.Kind() == slog.KindGroup {
		groupAttrs := attr.Value.Group()
		if len(attr.Key) == 0 {
			for _, groupAttr := range groupAttrs {
				h.writeAttr(writer, indent, groupAttr)
			}
		} else {
			if len(groupAttrs) == 0 {
				return
			}
			fmt.Fprintf(writer, "%sGroup: %s\n", indentStr, h.escape(attr.Key))
			for _, groupAttr := range groupAttrs {
				h.writeAttr(writer, indent+1, groupAttr)
			}
		}
	} else {
		fmt.Fprintf(writer, "%s%s:", indentStr, h.escape(attr.Key))
		valueStr := attr.Value.String()
		if len(valueStr) > 0 && bytes.ContainsRune([]byte(valueStr), '\n') {
			strings.SplitSeq(valueStr, "\n")(func(line string) bool {
				fmt.Fprintf(writer, "\n  %s%s", indentStr, h.escape(line))
				return true
			})
			writer.Write([]byte("\n"))
		} else {
			fmt.Fprintf(writer, " %s\n", h.escape(valueStr))
		}
	}
}

// Handle implements slog.Handler.Handle
func (h *ConsoleHandler) Handle(_ context.Context, record slog.Record) error {
	var buff bytes.Buffer

	if len(h.groups) > 0 {
		attrAny := make([]any, len(h.attrs))
		for i, attr := range h.attrs {
			attrAny[i] = attr
		}
		h.writeAttr(&buff, len(h.groups)-1, slog.Group(h.groups[len(h.groups)-1], attrAny...))
	} else {
		for _, attr := range h.attrs {
			h.writeAttr(&buff, 0, attr)
		}
	}

	indentStr := strings.Repeat("  ", len(h.groups))
	buff.Write([]byte(indentStr))

	if h.opts.Time && !record.Time.IsZero() {
		fmt.Fprintf(&buff, "%s ", record.Time.Round(0).Format(time.DateTime))
	}

	fmt.Fprintf(&buff, "%s %s\n", record.Level.String(), h.escape(record.Message))

	if h.opts.HandlerOptions.AddSource && record.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{record.PC})
		frame, _ := frames.Next()
		fmt.Fprintf(&buff, "%s  %s:%d", indentStr, frame.File, frame.Line)
		if len(frame.Function) > 0 {
			fmt.Fprintf(&buff, " (%s) ", frame.Function)
		} else {
			buff.Write([]byte(" "))
		}
		buff.Write([]byte("\n"))
	}

	if record.NumAttrs() > 0 {
		record.Attrs(func(attr slog.Attr) bool {
			h.writeAttr(&buff, len(h.groups)+1, attr)
			return true
		})
	}

	h.writer.Write(buff.Bytes())

	return nil
}
