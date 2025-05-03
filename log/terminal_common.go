package log

import (
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/fornellas/resonance/ansi"
	"github.com/fornellas/resonance/unicode"
)

// ANSI color scheme
type TerminalHandlerColorScheme struct {
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

var DefaultTerminalHandlerColorScheme = &TerminalHandlerColorScheme{
	GroupName:    ansi.SGRs{},
	AttrKey:      ansi.SGRs{ansi.FgCyan, ansi.Dim},
	AttrValue:    ansi.SGRs{ansi.Dim},
	Time:         ansi.SGRs{ansi.Dim},
	LevelDebug:   ansi.SGRs{ansi.FgCyan, ansi.Bold},
	MessageDebug: ansi.SGRs{ansi.Bold},
	LevelInfo:    ansi.SGRs{ansi.FgGreen, ansi.Bold},
	MessageInfo:  ansi.SGRs{ansi.Bold},
	LevelWarn:    ansi.SGRs{ansi.FgYellow, ansi.Bold},
	MessageWarn:  ansi.SGRs{ansi.Bold},
	LevelError:   ansi.SGRs{ansi.FgRed, ansi.Bold},
	MessageError: ansi.SGRs{ansi.Bold},
	File:         ansi.SGRs{ansi.Dim, ansi.FgBlue},
	Line:         ansi.SGRs{ansi.Dim, ansi.FgBlue},
	Function:     ansi.SGRs{ansi.Dim, ansi.FgBlue},
}

// TerminalHandlerOptions extends HandlerOptions with specific options.
type TerminalHandlerOptions struct {
	slog.HandlerOptions
	// Time layout for timestamps; if empty, time is not included in output.
	TimeLayout string
	// If true, force ANSI escape sequences for color, even when no TTY detected.
	ForceColor bool
	// If true, disable color, even when TTY detected; takes precedence over ForceColor.
	NoColor bool
	// ANSI color scheme. Default to DefaultColorScheme if unset.
	ColorScheme *TerminalHandlerColorScheme
}

func writeLevel(
	w io.Writer, colorScheme *TerminalHandlerColorScheme, level slog.Level,
) (int, error) {
	if level >= slog.LevelError {
		return colorScheme.LevelError.Fprintf(w, "%s", level.String())
	} else if level >= slog.LevelWarn {
		return colorScheme.LevelWarn.Fprintf(w, "%s", level.String())
	} else if level >= slog.LevelInfo {
		return colorScheme.LevelInfo.Fprintf(w, "%s", level.String())
	} else {
		return colorScheme.LevelDebug.Fprintf(w, "%s", level.String())
	}
}

func escape(s string) string {
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

func writeMessage(
	w io.Writer, colorScheme *TerminalHandlerColorScheme, level slog.Level, message string,
) (int, error) {
	message = escape(message)
	if level >= slog.LevelError {
		return colorScheme.MessageError.Fprintf(w, "%s", message)
	} else if level >= slog.LevelWarn {
		return colorScheme.MessageWarn.Fprintf(w, "%s", message)
	} else if level >= slog.LevelInfo {
		return colorScheme.MessageInfo.Fprintf(w, "%s", message)
	} else {
		return colorScheme.MessageDebug.Fprintf(w, "%s", message)
	}
}

func writeTime(
	w io.Writer,
	timeLayout string,
	t time.Time,
	colorScheme *TerminalHandlerColorScheme,
) (int, error) {
	if timeLayout != "" && !t.IsZero() {
		return colorScheme.Time.Fprintf(w, "%s", t.Round(0).Format(timeLayout))
	}
	return 0, nil
}

func writePC(
	w io.Writer,
	colorScheme *TerminalHandlerColorScheme,
	pc uintptr,
) error {
	if pc == 0 {
		return nil
	}
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	if _, err := colorScheme.File.Fprintf(w, "%s", frame.File); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, ":"); err != nil {
		return err
	}
	if _, err := colorScheme.Line.Fprintf(w, "%d", frame.Line); err != nil {
		return err
	}
	if len(frame.Function) > 0 {
		if _, err := fmt.Fprintf(w, " ("); err != nil {
			return err
		}
		if _, err := colorScheme.Function.Fprintf(w, "%s", frame.Function); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, ")"); err != nil {
			return err
		}
	}
	return nil
}

func writeGroup(
	w io.Writer,
	colorScheme *TerminalHandlerColorScheme,
	name string,
) error {
	emoji := ""
	r, _ := utf8.DecodeRuneInString(name)
	if !unicode.IsEmojiStartCodePoint(r) {
		emoji = "🏷️ "
	}
	if _, err := fmt.Fprintf(w, "%s", emoji); err != nil {
		return err
	}
	if _, err := colorScheme.GroupName.Fprintf(w, "%s", escape(name)); err != nil {
		return err
	}
	return nil
}
