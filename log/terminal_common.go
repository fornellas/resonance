package log

import (
	"io"
	"log/slog"
	"strconv"
	"time"

	"github.com/fornellas/resonance/ansi"
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
	LevelDebug:   ansi.SGRs{},
	MessageDebug: ansi.SGRs{ansi.FgCyan},
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
	w io.Writer, timeLayout string, t time.Time, colorScheme *TerminalHandlerColorScheme,
) (int, error) {
	if timeLayout != "" && !t.IsZero() {
		return colorScheme.Time.Fprintf(w, "%s", t.Round(0).Format(timeLayout))
	}
	return 0, nil
}
