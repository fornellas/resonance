// ANSI terminal escape sequences utilities.
package ansi

import (
	"bytes"
	"fmt"
	"io"
)

// Control Sequence Introducer
const CSI = "\033["

// Select Graphic Rendition display attribute.
type SGR uint

// String returns the ANSI escape sequence as a string for this SGR code.
// For example, SGR code 31 (FgRed) returns "\033[31m".
func (s SGR) String() string {
	return fmt.Sprintf("%s%dm", CSI, s)
}

// Sprintf works similar to fmt.Sprintf, but it wraps the formatted text with the SGR codes and
// the Reset code.
func (s SGR) Sprintf(format string, a ...any) string {
	a = append(
		[]any{
			s.String(),
		},
		append(a, Reset.String())...,
	)

	return fmt.Sprintf("%s"+format+"%s", a...)
}

// Fprintf works similar to fmt.Fprintf, but it wraps the formatted text with the SGR codes and
// the Reset code.
func (s SGR) Fprintf(w io.Writer, format string, a ...any) (int, error) {
	a = append(
		[]any{
			s.String(),
		},
		append(a, Reset.String())...,
	)
	return fmt.Fprintf(w, "%s"+format+"%s", a...)
}

const (
	Reset     SGR = 0
	Bold      SGR = 1
	Dim       SGR = 2
	Italic    SGR = 3
	Underline SGR = 4
	Blink     SGR = 5
	Reverse   SGR = 7
	Hidden    SGR = 8
	Strike    SGR = 9

	// Foreground colors (3-bit)
	FgBlack   SGR = 30
	FgRed     SGR = 31
	FgGreen   SGR = 32
	FgYellow  SGR = 33
	FgBlue    SGR = 34
	FgMagenta SGR = 35
	FgCyan    SGR = 36
	FgWhite   SGR = 37

	// Background colors (3-bit)
	BgBlack   SGR = 40
	BgRed     SGR = 41
	BgGreen   SGR = 42
	BgYellow  SGR = 43
	BgBlue    SGR = 44
	BgMagenta SGR = 45
	BgCyan    SGR = 46
	BgWhite   SGR = 47

	// Bright foreground colors (4-bit)
	FgDarkGray     SGR = 90
	FgLightRed     SGR = 91
	FgLightGreen   SGR = 92
	FgLightYellow  SGR = 93
	FgLightBlue    SGR = 94
	FgLightMagenta SGR = 95
	FgLightCyan    SGR = 96
	FgLightWhite   SGR = 97

	// Bright background colors (4-bit)
	BgDarkGray     SGR = 100
	BgLightRed     SGR = 101
	BgLightGreen   SGR = 102
	BgLightYellow  SGR = 103
	BgLightBlue    SGR = 104
	BgLightMagenta SGR = 105
	BgLightCyan    SGR = 106
	BgLightWhite   SGR = 107
)

// Select Graphic Rendition display attributes.
type SGRs []SGR

// String returns the ANSI escape sequence as a string for these SGR codes combined.
// For example, []SGR{FgRed, Bold} returns "\033[31;1m".
// If the SGRs slice is empty, it returns an empty string.
func (s SGRs) String() string {
	if len(s) == 0 {
		return ""
	}
	var buff bytes.Buffer
	buff.Write([]byte(CSI))
	for i, sgr := range s {
		if i+1 < len(s) {
			fmt.Fprintf(&buff, "%d;", sgr)
		} else {
			fmt.Fprintf(&buff, "%d", sgr)
		}
	}
	buff.WriteString("m")
	return buff.String()
}

// Sprintf works similar to fmt.Sprintf, but it wraps the formatted text with the SGR codes and
// the Reset code.
// If the SGRs slice is empty, it behaves as fmt.Sprintf.
func (s SGRs) Sprintf(format string, a ...any) string {
	if len(s) == 0 {
		return fmt.Sprintf(format, a...)
	}

	a = append(
		[]any{
			s.String(),
		},
		append(a, Reset.String())...,
	)
	return fmt.Sprintf("%s"+format+"%s", a...)
}

// Fprintf works similar to fmt.Fprintf, but it wraps the formatted text with the SGR codes and
// the Reset code.
// If the SGRs slice is empty, it behaves as fmt.Fprintf.
func (s SGRs) Fprintf(w io.Writer, format string, a ...any) (int, error) {
	if len(s) == 0 {
		return fmt.Fprintf(w, format, a...)
	}

	a = append(
		[]any{
			s.String(),
		},
		append(a, Reset.String())...,
	)
	return fmt.Fprintf(w, "%s"+format+"%s", a...)
}
