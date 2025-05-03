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
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/fornellas/resonance/unicode"
)

// currHandlerChain tracks the chain of TerminalTreeHandler instances to avoid
// duplicating output of handler attributes. It maintains the last written
// handler chain to determine which parts of the chain need to be written
// for subsequent log entries.
type currHandlerChain struct {
	chain []*TerminalTreeHandler
	m     sync.Mutex
}

func newCurrHandlerChain() *currHandlerChain {
	return &currHandlerChain{
		chain: []*TerminalTreeHandler{},
	}
}

func (s *currHandlerChain) writeHandlerGroupAttrs(writer io.Writer, handlerChain []*TerminalTreeHandler) {
	s.m.Lock()
	defer s.m.Unlock()

	for i, h := range handlerChain {
		if i+1 > len(s.chain) {
			h.writeHandlerGroupAttrs(writer, nil)
		} else {
			ch := s.chain[i]
			if h != ch {
				h.writeHandlerGroupAttrs(writer, ch)
			}
		}
	}

	s.chain = make([]*TerminalTreeHandler, len(handlerChain))
	copy(s.chain, handlerChain)
}

// TerminalTreeHandler implements slog.Handler interface with enhanced console output features.
// It provides colorized logging with level-appropriate colors, proper indentation for
// nested groups, and smart handling of multiline content. TerminalTreeHandler automatically
// detects terminal capabilities and enables or disables ANSI color codes accordingly,
// though this behavior can be overridden through options. The handler also supports
// customizable timestamp formats and sophisticated attribute formatting.
type TerminalTreeHandler struct {
	opts             *TerminalHandlerOptions
	writer           io.Writer
	writerMutex      *sync.Mutex
	color            bool
	groups           []string
	attrs            []slog.Attr
	handlerChain     []*TerminalTreeHandler
	currHandlerChain *currHandlerChain
}

// NewTerminalTreeHandler creates a new TerminalTreeHandler
func NewTerminalTreeHandler(w io.Writer, opts *TerminalHandlerOptions) *TerminalTreeHandler {
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

	color := !optsValue.NoColor && (optsValue.ForceColor || isTTY)
	if !color {
		optsValue.ColorScheme = &TerminalHandlerColorScheme{}
	}

	h := &TerminalTreeHandler{
		opts:             &optsValue,
		writer:           w,
		writerMutex:      &sync.Mutex{},
		color:            color,
		groups:           []string{},
		attrs:            []slog.Attr{},
		currHandlerChain: newCurrHandlerChain(),
	}
	h.handlerChain = []*TerminalTreeHandler{h}
	return h
}

// Enabled implements slog.Handler.Enabled
func (h *TerminalTreeHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *TerminalTreeHandler) clone() *TerminalTreeHandler {
	h2 := *h
	h2.groups = slices.Clip(h.groups)
	h2.attrs = slices.Clip(h.attrs)
	h2.handlerChain = slices.Clip(h.handlerChain)
	return &h2
}

// WithAttrs implements slog.Handler.WithAttrs
func (h *TerminalTreeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := h.clone()
	h2.attrs = append(h2.attrs, attrs...)
	h2.handlerChain[len(h2.handlerChain)-1] = h2
	return h2
}

// WithGroup implements slog.Handler.WithGroup
func (h *TerminalTreeHandler) WithGroup(name string) slog.Handler {
	if len(name) == 0 {
		return h
	}

	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	h2.attrs = []slog.Attr{}
	h2.handlerChain = append(h2.handlerChain, h2)
	return h2
}

func (h *TerminalTreeHandler) writeAttr(writer io.Writer, indent int, attr slog.Attr) {
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
			emoji := ""
			r, _ := utf8.DecodeRuneInString(attr.Key)
			if !unicode.IsEmojiStartCodePoint(r) {
				emoji = "🏷️ "
			}
			fmt.Fprintf(writer, "%s%s%s\n", indentStr, emoji, h.opts.ColorScheme.GroupName.Sprintf(
				"%s", escape(attr.Key),
			))
			for _, groupAttr := range groupAttrs {
				h.writeAttr(writer, indent+1, groupAttr)
			}
		}
	} else {
		fmt.Fprintf(writer, "%s%s:", indentStr, h.opts.ColorScheme.AttrKey.Sprintf(
			"%s", escape(attr.Key),
		))
		valueStr := attr.Value.String()
		if len(valueStr) > 0 && bytes.ContainsRune([]byte(valueStr), '\n') {
			strings.SplitSeq(valueStr, "\n")(func(line string) bool {
				fmt.Fprintf(writer, "\n  %s%s", indentStr, h.opts.ColorScheme.AttrValue.Sprintf(
					"%s", escape(line),
				))
				return true
			})
			writer.Write([]byte("\n"))
		} else {
			fmt.Fprintf(writer, " %s\n", h.opts.ColorScheme.AttrValue.Sprintf(
				"%s", escape(valueStr),
			))
		}
	}
}

func (h *TerminalTreeHandler) sameGroups(h2 *TerminalTreeHandler) bool {
	if len(h.groups) != len(h2.groups) {
		return false
	}
	for i, v := range h.groups {
		if v != h2.groups[i] {
			return false
		}
	}
	return true
}

// write handler group & attrs, as a function of the current handler at the chain, preventing
// duplicate attrs
func (h *TerminalTreeHandler) writeHandlerGroupAttrs(writer io.Writer, ch *TerminalTreeHandler) {
	var attrs []slog.Attr
	var sameGroups bool
	if ch != nil {
		if sameGroups = h.sameGroups(ch); sameGroups {
			attrs = []slog.Attr{}
			for i, attr := range h.attrs {
				if i+1 <= len(ch.attrs) && attr.Equal(ch.attrs[i]) {
					continue
				}
				attrs = append(attrs, attr)
			}
		} else {
			attrs = h.attrs
		}
	} else {
		attrs = h.attrs
	}
	if len(h.groups) > 0 {
		if sameGroups {
			for _, attr := range attrs {
				h.writeAttr(writer, len(h.groups), attr)
			}
		} else {
			attrAny := make([]any, len(attrs))
			for i, attr := range attrs {
				attrAny[i] = attr
			}
			h.writeAttr(writer, len(h.groups)-1, slog.Group(h.groups[len(h.groups)-1], attrAny...))
		}
	} else {
		for _, attr := range attrs {
			h.writeAttr(writer, 0, attr)
		}
	}
}

func (h *TerminalTreeHandler) writeLevelMessage(
	w io.Writer, level slog.Level, message string,
) (int, error) {
	var n int

	np, err := writeLevel(w, h.color, h.opts.ColorScheme, level)
	n += np
	if err != nil {
		return n, err
	}

	np, err = w.Write([]byte(" "))
	n += np
	if err != nil {
		return n, err
	}

	np, err = writeMessage(w, h.opts.ColorScheme, level, message)
	n += np
	if err != nil {
		return n, err
	}

	np, err = w.Write([]byte("\n"))
	n += np
	if err != nil {
		return n, err
	}

	return n, nil
}

// Handle implements slog.Handler.Handle
func (h *TerminalTreeHandler) Handle(_ context.Context, record slog.Record) error {
	var buff bytes.Buffer

	h.currHandlerChain.writeHandlerGroupAttrs(&buff, h.handlerChain)

	indentStr := strings.Repeat("  ", len(h.groups))
	buff.Write([]byte(indentStr))

	if h.opts.TimeLayout != "" && !record.Time.IsZero() {
		timeStr := record.Time.Round(0).Format(h.opts.TimeLayout)
		fmt.Fprintf(&buff, "%s ", h.opts.ColorScheme.AttrValue.Sprintf(
			"%s", timeStr,
		))
	}

	h.writeLevelMessage(&buff, record.Level, record.Message)

	if h.opts.HandlerOptions.AddSource && record.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{record.PC})
		frame, _ := frames.Next()

		fmt.Fprintf(&buff, "%s  %s:%s", indentStr,
			h.opts.ColorScheme.File.Sprintf("%s", frame.File),
			h.opts.ColorScheme.Line.Sprintf("%d", frame.Line),
		)
		if len(frame.Function) > 0 {
			fmt.Fprintf(&buff, " (%s)", h.opts.ColorScheme.Function.Sprintf("%s", frame.Function))
		}
		buff.Write([]byte("\n"))
	}

	if record.NumAttrs() > 0 {
		record.Attrs(func(attr slog.Attr) bool {
			h.writeAttr(&buff, len(h.groups)+1, attr)
			return true
		})
	}

	h.writerMutex.Lock()
	defer h.writerMutex.Unlock()
	h.writer.Write(buff.Bytes())

	return nil
}
