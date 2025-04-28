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
	"sync"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/fornellas/resonance/ansi"
	"github.com/fornellas/resonance/unicode"
)

// currHandlerChain tracks the chain of ConsoleHandler instances to avoid
// duplicating output of handler attributes. It maintains the last written
// handler chain to determine which parts of the chain need to be written
// for subsequent log entries.
type currHandlerChain struct {
	chain []*ConsoleHandler
	m     sync.Mutex
}

func newCurrHandlerChain() *currHandlerChain {
	return &currHandlerChain{
		chain: []*ConsoleHandler{},
	}
}

func (s *currHandlerChain) writeHandlerGroupAttrs(writer io.Writer, handlerChain []*ConsoleHandler) {
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

	s.chain = make([]*ConsoleHandler, len(handlerChain))
	copy(s.chain, handlerChain)
}

// ANSI color scheme
type ColorScheme struct {
	GroupName    ansi.SGRs
	AttrKey      ansi.SGRs
	AttrValue    ansi.SGRs
	Time         ansi.SGRs
	LevelDebug   ansi.SGRs
	MessageDebug ansi.SGRs
	LevelInfo    ansi.SGRs
	MessageInfo  ansi.SGRs
	LevelWarn    ansi.SGRs
	MessageWarn  ansi.SGRs
	LevelError   ansi.SGRs
	MessageError ansi.SGRs
	File         ansi.SGRs
	Line         ansi.SGRs
	Function     ansi.SGRs
}

var DefaultColorScheme = &ColorScheme{
	GroupName:    ansi.SGRs{},
	AttrKey:      ansi.SGRs{ansi.FgCyan, ansi.Dim},
	AttrValue:    ansi.SGRs{ansi.Dim},
	Time:         ansi.SGRs{ansi.Dim},
	LevelDebug:   ansi.SGRs{},
	MessageDebug: ansi.SGRs{ansi.Dim},
	LevelInfo:    ansi.SGRs{},
	MessageInfo:  ansi.SGRs{ansi.Bold},
	LevelWarn:    ansi.SGRs{ansi.FgYellow, ansi.Bold},
	MessageWarn:  ansi.SGRs{ansi.Bold},
	LevelError:   ansi.SGRs{ansi.FgRed, ansi.Bold},
	MessageError: ansi.SGRs{ansi.Bold},
	File:         ansi.SGRs{ansi.Dim},
	Line:         ansi.SGRs{ansi.Dim},
	Function:     ansi.SGRs{ansi.Dim},
}

// ConsoleHandlerOptions extends HandlerOptions with ConsoleHandler specific options.
type ConsoleHandlerOptions struct {
	slog.HandlerOptions
	// Time layout for timestamps; if empty, time is not included in output.
	TimeLayout string
	// If true, force ANSI escape sequences for color, even when no TTY detected.
	ForceColor bool
	// If true, disable color, even when TTY detected; takes precedence over ForceColor.
	NoColor bool
	// ANSI color scheme. Default to DefaultColorScheme if unset.
	ColorScheme *ColorScheme
}

// ConsoleHandler implements slog.Handler interface with enhanced console output features.
// It provides colorized logging with level-appropriate colors, proper indentation for
// nested groups, and smart handling of multiline content. ConsoleHandler automatically
// detects terminal capabilities and enables or disables ANSI color codes accordingly,
// though this behavior can be overridden through options. The handler also supports
// customizable timestamp formats and sophisticated attribute formatting.
type ConsoleHandler struct {
	opts             *ConsoleHandlerOptions
	writer           io.Writer
	writerMutex      *sync.Mutex
	color            bool
	groups           []string
	attrs            []slog.Attr
	handlerChain     []*ConsoleHandler
	currHandlerChain *currHandlerChain
}

// NewConsoleHandler creates a new ConsoleHandler
func NewConsoleHandler(w io.Writer, opts *ConsoleHandlerOptions) *ConsoleHandler {
	var optsValue ConsoleHandlerOptions
	if opts != nil {
		optsValue = *opts
	}

	if optsValue.ColorScheme == nil {
		optsValue.ColorScheme = DefaultColorScheme
	}

	isTTY := false
	if f, ok := w.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	color := !optsValue.NoColor && (optsValue.ForceColor || isTTY)
	if !color {
		optsValue.ColorScheme = &ColorScheme{}
	}

	h := &ConsoleHandler{
		opts:             &optsValue,
		writer:           w,
		writerMutex:      &sync.Mutex{},
		color:            color,
		groups:           []string{},
		attrs:            []slog.Attr{},
		currHandlerChain: newCurrHandlerChain(),
	}
	h.handlerChain = []*ConsoleHandler{h}
	return h
}

func (h *ConsoleHandler) clone() *ConsoleHandler {
	h2 := *h
	h2.groups = slices.Clip(h.groups)
	h2.attrs = slices.Clip(h.attrs)
	h2.handlerChain = slices.Clip(h.handlerChain)
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
	h2.handlerChain[len(h2.handlerChain)-1] = h2
	return h2
}

// WithGroup implements slog.Handler.WithGroup
func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	if len(name) == 0 {
		return h
	}

	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	h2.attrs = []slog.Attr{}
	h2.handlerChain = append(h2.handlerChain, h2)
	return h2
}

func (h *ConsoleHandler) escape(s string) string {
	rs := []rune{}
	for _, r := range s {
		if r == '\t' || strconv.IsPrint(r) {
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
			emoji := ""
			r, _ := utf8.DecodeRuneInString(attr.Key)
			if !unicode.IsEmojiStartCodePoint(r) {
				emoji = "🏷️ "
			}
			fmt.Fprintf(writer, "%s%s%s\n", indentStr, emoji, h.opts.ColorScheme.GroupName.Sprintf(
				"%s", h.escape(attr.Key),
			))
			for _, groupAttr := range groupAttrs {
				h.writeAttr(writer, indent+1, groupAttr)
			}
		}
	} else {
		fmt.Fprintf(writer, "%s%s:", indentStr, h.opts.ColorScheme.AttrKey.Sprintf(
			"%s", h.escape(attr.Key),
		))
		valueStr := attr.Value.String()
		if len(valueStr) > 0 && bytes.ContainsRune([]byte(valueStr), '\n') {
			strings.SplitSeq(valueStr, "\n")(func(line string) bool {
				fmt.Fprintf(writer, "\n  %s%s", indentStr, h.opts.ColorScheme.AttrValue.Sprintf(
					"%s", h.escape(line),
				))
				return true
			})
			writer.Write([]byte("\n"))
		} else {
			fmt.Fprintf(writer, " %s\n", h.opts.ColorScheme.AttrValue.Sprintf(
				"%s", h.escape(valueStr),
			))
		}
	}
}

func (h *ConsoleHandler) sameGroups(h2 *ConsoleHandler) bool {
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
func (h *ConsoleHandler) writeHandlerGroupAttrs(writer io.Writer, ch *ConsoleHandler) {
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

func (h *ConsoleHandler) printLevelMessage(w io.Writer, level slog.Level, message string) {
	message = h.escape(message)
	if level >= slog.LevelError {
		if len(h.opts.ColorScheme.LevelError) > 0 || !h.color {
			fmt.Fprintf(w, "%s ", h.opts.ColorScheme.LevelError.Sprintf(
				"%s", level.String(),
			))
		}
		fmt.Fprintf(w, "%s\n", h.opts.ColorScheme.MessageError.Sprintf("%s", message))
	} else if level >= slog.LevelWarn {
		if len(h.opts.ColorScheme.LevelWarn) > 0 || !h.color {
			fmt.Fprintf(w, "%s ", h.opts.ColorScheme.LevelWarn.Sprintf(
				"%s", level.String(),
			))
		}
		fmt.Fprintf(w, "%s\n", h.opts.ColorScheme.MessageWarn.Sprintf("%s", message))
	} else if level >= slog.LevelInfo {
		if len(h.opts.ColorScheme.LevelInfo) > 0 || !h.color {
			fmt.Fprintf(w, "%s ", h.opts.ColorScheme.LevelInfo.Sprintf(
				"%s", level.String(),
			))
		}
		fmt.Fprintf(w, "%s\n", h.opts.ColorScheme.MessageInfo.Sprintf("%s", message))
	} else {
		if len(h.opts.ColorScheme.LevelDebug) > 0 || !h.color {
			fmt.Fprintf(w, "%s ", h.opts.ColorScheme.LevelDebug.Sprintf(
				"%s", level.String(),
			))
		}
		fmt.Fprintf(w, "%s\n", h.opts.ColorScheme.MessageDebug.Sprintf("%s", message))
	}
}

// Handle implements slog.Handler.Handle
func (h *ConsoleHandler) Handle(_ context.Context, record slog.Record) error {
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

	h.printLevelMessage(&buff, record.Level, record.Message)

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
