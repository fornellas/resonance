package log

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"

	"golang.org/x/term"
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

func (s *currHandlerChain) writeHandlerGroupAttrs(writer io.Writer, handlerChain []*TerminalTreeHandler) error {
	s.m.Lock()
	defer s.m.Unlock()

	for i, h := range handlerChain {
		if i+1 > len(s.chain) {
			if err := h.writeHandlerGroupAttrs(writer, nil); err != nil {
				return err
			}
		} else {
			ch := s.chain[i]
			if h != ch {
				if err := h.writeHandlerGroupAttrs(writer, ch); err != nil {
					return err
				}
			}
		}
	}

	s.chain = make([]*TerminalTreeHandler, len(handlerChain))
	copy(s.chain, handlerChain)
	return nil
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

	if !(!optsValue.NoColor && (optsValue.ForceColor || isTTY)) {
		optsValue.ColorScheme = &TerminalHandlerColorScheme{}
	}

	h := &TerminalTreeHandler{
		opts:             &optsValue,
		writer:           w,
		writerMutex:      &sync.Mutex{},
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
	h2.groups = slices.Clone(h.groups)
	h2.attrs = slices.Clone(h.attrs)
	h2.handlerChain = slices.Clone(h.handlerChain)
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

func (h *TerminalTreeHandler) writeAttrGroupValue(w io.Writer, indent int, attr slog.Attr) error {
	groupAttrs := attr.Value.Group()
	if len(attr.Key) == 0 {
		for _, groupAttr := range groupAttrs {
			if err := h.writeAttr(w, indent, groupAttr); err != nil {
				return err
			}
		}
	} else {
		indentStr := strings.Repeat("  ", indent)
		if _, err := w.Write([]byte(indentStr)); err != nil {
			return err
		}
		if _, err := writeGroup(w, h.opts.ColorScheme, attr.Key); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			return err
		}
		for _, groupAttr := range groupAttrs {
			if err := h.writeAttr(w, indent+1, groupAttr); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *TerminalTreeHandler) writeAttrNonGroupValue(w io.Writer, indent int, attr slog.Attr) error {
	indentStr := strings.Repeat("  ", indent)
	if _, err := fmt.Fprintf(w, "%s", indentStr); err != nil {
		return err
	}
	if _, err := h.opts.ColorScheme.AttrKey.Fprintf(w, "%s:", escape(attr.Key)); err != nil {
		return err
	}
	valueStr := attr.Value.String()
	if len(valueStr) > 0 && bytes.ContainsRune([]byte(valueStr), '\n') {
		var fnErr error
		strings.SplitSeq(valueStr, "\n")(func(line string) bool {
			if _, fnErr = fmt.Fprintf(w, "\n  %s", indentStr); fnErr != nil {
				return false
			}
			_, fnErr = h.opts.ColorScheme.AttrValue.Fprintf(w, "%s", escape(line))
			return fnErr == nil
		})
		if fnErr != nil {
			return fnErr
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			return err
		}
	} else {
		if _, err := h.opts.ColorScheme.AttrValue.Fprintf(w, " %s", escape(valueStr)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "\n"); err != nil {
			return err
		}
	}
	return nil
}

func (h *TerminalTreeHandler) writeAttr(w io.Writer, indent int, attr slog.Attr) error {
	attr.Value = attr.Value.Resolve()
	if h.opts.ReplaceAttr != nil && attr.Value.Kind() != slog.KindGroup {
		attr = h.opts.ReplaceAttr(h.groups, attr)
		attr.Value = attr.Value.Resolve()
	}

	if attr.Equal(slog.Attr{}) {
		return nil
	}

	if attr.Value.Kind() == slog.KindGroup {
		if err := h.writeAttrGroupValue(w, indent, attr); err != nil {
			return err
		}
	} else {
		if err := h.writeAttrNonGroupValue(w, indent, attr); err != nil {
			return err
		}
	}

	return nil
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
func (h *TerminalTreeHandler) writeHandlerGroupAttrs(writer io.Writer, ch *TerminalTreeHandler) error {
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
				if err := h.writeAttr(writer, len(h.groups), attr); err != nil {
					return err
				}
			}
		} else {
			attrAny := make([]any, len(attrs))
			for i, attr := range attrs {
				attrAny[i] = attr
			}
			if err := h.writeAttr(writer, len(h.groups)-1, slog.Group(h.groups[len(h.groups)-1], attrAny...)); err != nil {
				return err
			}
		}
	} else {
		for _, attr := range attrs {
			if err := h.writeAttr(writer, 0, attr); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *TerminalTreeHandler) writeLevelMessage(
	w io.Writer, level slog.Level, message string,
) (int, error) {
	var n int

	np, err := writeLevel(w, h.opts.ColorScheme, level)
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

	// Handler: Group + Attr
	if err := h.currHandlerChain.writeHandlerGroupAttrs(&buff, h.handlerChain); err != nil {
		return err
	}

	// Indent
	if _, err := buff.WriteString(strings.Repeat("  ", len(h.groups))); err != nil {
		return err
	}

	// Record: Level + Message
	if _, err := h.writeLevelMessage(&buff, record.Level, record.Message); err != nil {
		return err
	}

	// Record: Time
	if h.opts.TimeLayout != "" && !record.Time.IsZero() {
		if _, err := buff.WriteString(strings.Repeat("  ", len(h.groups)+1)); err != nil {
			return err
		}
		if _, err := writeTime(&buff, h.opts.TimeLayout, record.Time, h.opts.ColorScheme); err != nil {
			return err
		}
		if _, err := buff.WriteString("\n"); err != nil {
			return err
		}
	}

	// Record: PC
	if h.opts.HandlerOptions.AddSource {
		if _, err := fmt.Fprintf(&buff, "%s  ", strings.Repeat("  ", len(h.groups))); err != nil {
			return err
		}
		writePC(&buff, h.opts.ColorScheme, record.PC)
		if _, err := buff.WriteString("\n"); err != nil {
			return err
		}
	}

	// Record: Attr
	if record.NumAttrs() > 0 {
		var attrErr error
		record.Attrs(func(attr slog.Attr) bool {
			attrErr = h.writeAttr(&buff, len(h.groups)+1, attr)
			return attrErr == nil
		})
		if attrErr != nil {
			return attrErr
		}
	}

	// Flush
	h.writerMutex.Lock()
	defer h.writerMutex.Unlock()
	_, err := h.writer.Write(buff.Bytes())
	return err
}
