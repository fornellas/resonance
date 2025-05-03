package log

import (
	"log/slog"

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
