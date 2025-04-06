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

	"golang.org/x/term"
)

// ANSI color codes for console output
const (
	reset     = "\033[0m"
	bold      = "\033[1m"
	dim       = "\033[2m"
	italic    = "\033[3m"
	underline = "\033[4m"
	blink     = "\033[5m"
	reverse   = "\033[7m"
	hidden    = "\033[8m"
	strike    = "\033[9m"

	// Foreground colors (3-bit)
	black   = "\033[30m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"

	// Background colors (3-bit)
	// blackBg   = "\033[40m"
	// redBg     = "\033[41m"
	// greenBg   = "\033[42m"
	// yellowBg  = "\033[43m"
	// blueBg    = "\033[44m"
	// magentaBg = "\033[45m"
	// cyanBg    = "\033[46m"
	// whiteBg   = "\033[47m"

	// // Bright foreground colors (4-bit)
	// darkGray     = "\033[90m"
	// lightRed     = "\033[91m"
	// lightGreen   = "\033[92m"
	// lightYellow  = "\033[93m"
	// lightBlue    = "\033[94m"
	// lightMagenta = "\033[95m"
	// lightCyan    = "\033[96m"
	// lightWhite   = "\033[97m"

	// // Bright background colors (4-bit)
	// darkGrayBg     = "\033[100m"
	// lightRedBg     = "\033[101m"
	// lightGreenBg   = "\033[102m"
	// lightYellowBg  = "\033[103m"
	// lightBlueBg    = "\033[104m"
	// lightMagentaBg = "\033[105m"
	// lightCyanBg    = "\033[106m"
	// lightWhiteBg   = "\033[107m"
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
		if i+1 > len(s.chain) || h != (s.chain)[i] {
			h.writeHandlerGroupAttrs(writer)
		}
	}

	s.chain = make([]*ConsoleHandler, len(handlerChain))
	copy(s.chain, handlerChain)
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
	if opts == nil {
		opts = &ConsoleHandlerOptions{}
	}

	isTTY := false
	if f, ok := w.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	h := &ConsoleHandler{
		opts:             opts,
		writer:           w,
		writerMutex:      &sync.Mutex{},
		color:            !opts.NoColor && (opts.ForceColor || isTTY),
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
	h2.handlerChain = append(h2.handlerChain, h2)
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

func (h *ConsoleHandler) colorize(s string, color string) string {
	if h.color {
		return color + s + reset
	}
	return s
}

func (h *ConsoleHandler) levelColor(level slog.Level) string {
	if !h.color {
		return ""
	}

	switch {
	case level >= slog.LevelError:
		return red
	case level >= slog.LevelWarn:
		return yellow
	case level >= slog.LevelInfo:
		return green
	case level >= slog.LevelDebug:
		return cyan
	default:
		return blue
	}
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
			fmt.Fprintf(writer, "%s%s: %s\n", indentStr, h.colorize("Group", blue+bold), h.colorize(h.escape(attr.Key), bold))
			for _, groupAttr := range groupAttrs {
				h.writeAttr(writer, indent+1, groupAttr)
			}
		}
	} else {
		fmt.Fprintf(writer, "%s%s:", indentStr, h.colorize(h.escape(attr.Key), cyan+dim))
		valueStr := attr.Value.String()
		if len(valueStr) > 0 && bytes.ContainsRune([]byte(valueStr), '\n') {
			strings.SplitSeq(valueStr, "\n")(func(line string) bool {
				fmt.Fprintf(writer, "\n  %s%s", indentStr, h.colorize(h.escape(line), dim))
				return true
			})
			writer.Write([]byte("\n"))
		} else {
			fmt.Fprintf(writer, " %s\n", h.colorize(h.escape(valueStr), dim))
		}
	}
}

func (h *ConsoleHandler) writeHandlerGroupAttrs(writer io.Writer) {
	if len(h.groups) > 0 {
		attrAny := make([]any, len(h.attrs))
		for i, attr := range h.attrs {
			attrAny[i] = attr
		}
		h.writeAttr(writer, len(h.groups)-1, slog.Group(h.groups[len(h.groups)-1], attrAny...))
	} else {
		for _, attr := range h.attrs {
			h.writeAttr(writer, 0, attr)
		}
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
		fmt.Fprintf(&buff, "%s ", h.colorize(timeStr, dim))
	}

	levelColor := h.levelColor(record.Level)
	levelStr := h.colorize(record.Level.String(), levelColor+bold)
	fmt.Fprintf(&buff, "%s %s\n", levelStr, h.colorize(h.escape(record.Message), bold))

	if h.opts.HandlerOptions.AddSource && record.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{record.PC})
		frame, _ := frames.Next()
		fileInfo := fmt.Sprintf("%s:%d", frame.File, frame.Line)
		fmt.Fprintf(&buff, "%s  %s", indentStr, h.colorize(fileInfo, dim))
		if len(frame.Function) > 0 {
			fmt.Fprintf(&buff, " (%s)", h.colorize(frame.Function, dim))
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
