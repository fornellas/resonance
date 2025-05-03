package log

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"slices"
	"sync"

	"golang.org/x/term"
)

type terminalLineHandlerAttrWriter struct {
	colorScheme *TerminalHandlerColorScheme
	replaceAttr func(groups []string, a slog.Attr) slog.Attr
}

func (aw *terminalLineHandlerAttrWriter) writeAttrGroupValue(
	w io.Writer,
	groups []string,
	attr slog.Attr,
) (int, error) {
	var n, nt int
	var err error

	attrs := attr.Value.Group()
	if len(attr.Key) == 0 {
		for _, groupAttr := range attrs {
			if n, err = aw.writeAttr(
				w,
				groups,
				groupAttr,
			); err != nil {
				return nt + n, err
			}
			nt += n
		}
	} else {
		ga := &groupAttrs{
			ColorScheme: aw.colorScheme,
			ReplaceAttr: aw.replaceAttr,
			Group:       attr.Key,
			Groups:      append(groups, attr.Key),
			Attrs:       attrs,
		}

		if n, err = ga.write(w); err != nil {
			return nt + n, err
		}
		nt += n
	}

	return nt, nil
}

func (aw *terminalLineHandlerAttrWriter) writeAttr(
	w io.Writer,
	groups []string,
	attr slog.Attr,
) (int, error) {
	attr.Value = attr.Value.Resolve()
	if aw.replaceAttr != nil && attr.Value.Kind() != slog.KindGroup {
		attr = aw.replaceAttr(groups, attr)
		attr.Value = attr.Value.Resolve()
	}

	if attr.Equal(slog.Attr{}) {
		return 0, nil
	}

	var n, nt int
	var err error

	if attr.Value.Kind() == slog.KindGroup {
		if n, err = aw.writeAttrGroupValue(
			w,
			groups,
			attr,
		); err != nil {
			return nt + n, err
		}
		nt += n
	} else {
		if n, err = aw.colorScheme.AttrKey.Fprintf(w, "%s", escape(attr.Key)); err != nil {
			return nt + n, err
		}
		nt += n

		if n, err = w.Write([]byte(": ")); err != nil {
			return nt + n, err
		}

		if n, err = aw.colorScheme.AttrValue.Fprintf(w, "%s", escape(attr.Value.String())); err != nil {
			return nt + n, err
		}
		nt += n
	}

	return nt, err
}

func (aw *terminalLineHandlerAttrWriter) writeAttrs(
	w io.Writer,
	groups []string,
	attrs []slog.Attr,
) (int, error) {
	if len(attrs) == 0 {
		return 0, nil
	}

	var n, nt int
	var err error

	if n, err = w.Write([]byte("[")); err != nil {
		return nt + n, err
	}
	nt += n

	for i, attr := range attrs {
		if i > 0 {
			if n, err = w.Write([]byte(", ")); err != nil {
				return nt + n, err
			}
			nt += n
		}

		if n, err := aw.writeAttr(w, groups, attr); err != nil {
			return nt + n, err
		}
		nt += n
	}

	if n, err = w.Write([]byte("]")); err != nil {
		return nt + n, err
	}
	nt += n

	return nt, nil
}

type groupAttrs struct {
	ColorScheme *TerminalHandlerColorScheme
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
	Group       string
	Groups      []string
	Attrs       []slog.Attr
}

func (ga *groupAttrs) write(w io.Writer) (int, error) {
	var n, nt int
	var err error

	if len(ga.Group) > 0 {
		if n, err = writeGroup(w, ga.ColorScheme, ga.Group); err != nil {
			return n, err
		}
		nt += n
	}

	if len(ga.Attrs) > 0 {
		if nt > 0 {
			if n, err = w.Write([]byte(" ")); err != nil {
				return nt + n, err
			}
			nt += n
		}

		attrWriter := &terminalLineHandlerAttrWriter{
			colorScheme: ga.ColorScheme,
			replaceAttr: ga.ReplaceAttr,
		}
		if n, err = attrWriter.writeAttrs(
			w,
			ga.Groups,
			ga.Attrs,
		); err != nil {
			return nt + n, err
		}
		nt += n
	}

	return nt, nil
}

// TerminalLineHandler is a slog.Handler implementation that formats log records
// as lines of text suitable for terminal output, with optional colorization.
// It provides a structured, human-readable output format with customizable styling
// through color schemes.
//
// Features:
// - Terminal-aware colorized output (auto-detects terminal capabilities)
// - Custom time formatting
// - Formatted level indicators
// - Hierarchical group support
// - Structured attribute rendering
// - Source code location reporting (when enabled)
//
// The handler will automatically detect if the output is a terminal and
// enable/disable colors accordingly, unless explicitly configured otherwise.
type TerminalLineHandler struct {
	opts        *TerminalHandlerOptions
	writer      io.Writer
	writerMutex *sync.Mutex
	groupAttrs  []groupAttrs
}

// NewTerminalLineHandler creates a new TerminalTextHandler
func NewTerminalLineHandler(w io.Writer, opts *TerminalHandlerOptions) *TerminalLineHandler {
	var optsValue TerminalHandlerOptions
	if opts != nil {
		optsValue = *opts
	}

	if optsValue.ColorScheme == nil {
		optsValue.ColorScheme = DefaultTerminalHandlerColorScheme
	}

	isTTY := false
	if f, ok := w.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	if !(!optsValue.NoColor && (optsValue.ForceColor || isTTY)) {
		optsValue.ColorScheme = &TerminalHandlerColorScheme{}
	}

	return &TerminalLineHandler{
		opts:        &optsValue,
		writer:      w,
		writerMutex: &sync.Mutex{},
		groupAttrs: []groupAttrs{
			groupAttrs{
				ColorScheme: optsValue.ColorScheme,
				ReplaceAttr: optsValue.ReplaceAttr,
			},
		},
	}
}

func (h *TerminalLineHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *TerminalLineHandler) groups() []string {
	var groups []string
	for _, ga := range h.groupAttrs {
		groups = append(groups, ga.Group)
	}
	return groups
}

//gocyclo:ignore
func (h *TerminalLineHandler) Handle(ctx context.Context, record slog.Record) error {
	var buff bytes.Buffer
	var n int
	var err error

	// Record: Time
	if n, err = writeTime(&buff, h.opts.TimeLayout, record.Time, h.opts.ColorScheme); err != nil {
		return err
	} else if n > 0 {
		if _, err = buff.WriteString(" "); err != nil {
			return err
		}
	}

	// Record: Level
	if n, err = writeLevel(&buff, h.opts.ColorScheme, record.Level); err != nil {
		return err
	} else if n > 0 {
		if _, err = buff.WriteString(" "); err != nil {
			return err
		}
	}

	// Handler: Groups + Attrs
	nt := 0
	for i, ga := range h.groupAttrs {
		if i > 0 {
			if n, err = buff.WriteString(" > "); err != nil {
				return err
			}
			nt += n
		}
		n, err = ga.write(&buff)
		if err != nil {
			return err
		}
		nt += n
	}
	if nt > 0 {
		if _, err = buff.WriteString(": "); err != nil {
			return err
		}
	}

	// Record: Message
	if _, err = writeMessage(&buff, h.opts.ColorScheme, record.Level, record.Message); err != nil {
		return err
	}

	// Record: Attr
	if record.NumAttrs() > 0 {
		if _, err = buff.WriteString(" "); err != nil {
			return err
		}

		attrs := []slog.Attr{}
		record.Attrs(func(attr slog.Attr) bool {
			attrs = append(attrs, attr)
			return true
		})

		attrWriter := &terminalLineHandlerAttrWriter{
			colorScheme: h.opts.ColorScheme,
			replaceAttr: h.opts.ReplaceAttr,
		}
		if _, err := attrWriter.writeAttrs(
			&buff,
			h.groups(),
			attrs,
		); err != nil {
			return err
		}
	}

	// Record: PC
	if h.opts.HandlerOptions.AddSource {
		if _, err := buff.WriteString(" "); err != nil {
			return err
		}
		writePC(&buff, h.opts.ColorScheme, record.PC)
	}

	// New line
	if _, err = buff.WriteString("\n"); err != nil {
		return err
	}

	// Flush
	h.writerMutex.Lock()
	defer h.writerMutex.Unlock()
	_, err = h.writer.Write(buff.Bytes())
	return err
}

func (h *TerminalLineHandler) clone() *TerminalLineHandler {
	h2 := *h
	h2.groupAttrs = slices.Clone(h.groupAttrs)
	return &h2
}

func (h *TerminalLineHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := h.clone()
	h2.groupAttrs[len(h2.groupAttrs)-1].Attrs = append(
		h2.groupAttrs[len(h2.groupAttrs)-1].Attrs,
		attrs...,
	)
	return h2
}

func (h *TerminalLineHandler) WithGroup(name string) slog.Handler {
	if len(name) == 0 {
		return h
	}

	h2 := h.clone()
	lastGroupAttrs := &h2.groupAttrs[len(h2.groupAttrs)-1]
	if len(lastGroupAttrs.Group) == 0 && len(lastGroupAttrs.Attrs) == 0 {
		lastGroupAttrs.Group = name
	} else {
		h2.groupAttrs = append(h2.groupAttrs, groupAttrs{
			ColorScheme: h2.opts.ColorScheme,
			ReplaceAttr: h2.opts.ReplaceAttr,
			Group:       name,
			Groups:      append(h2.groups(), name),
		})
	}

	return h2
}
