package ansi

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSGR(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		tests := []struct {
			sgr      SGR
			expected string
		}{
			{FgRed, "\033[31m"},
			{Bold, "\033[1m"},
			{Reset, "\033[0m"},
			{BgGreen, "\033[42m"},
			{FgLightBlue, "\033[94m"},
		}

		for _, test := range tests {
			require.Equal(t, test.expected, test.sgr.String())
		}
	})

	t.Run("Sprintf", func(t *testing.T) {
		tests := []struct {
			sgr      SGR
			format   string
			args     []any
			expected string
		}{
			{FgRed, "Hello", nil, "\033[31mHello\033[0m"},
			{Bold, "Number: %d", []any{42}, "\033[1mNumber: 42\033[0m"},
			{FgGreen, "%s: %d", []any{"Count", 5}, "\033[32mCount: 5\033[0m"},
		}

		for _, test := range tests {
			require.Equal(t, test.expected, test.sgr.Sprintf(test.format, test.args...))
		}
	})

	t.Run("Fprintf", func(t *testing.T) {
		tests := []struct {
			sgr      SGR
			format   string
			args     []any
			expected string
		}{
			{FgRed, "Hello", nil, "\033[31mHello\033[0m"},
			{Bold, "Number: %d", []any{42}, "\033[1mNumber: 42\033[0m"},
			{FgGreen, "%s: %d", []any{"Count", 5}, "\033[32mCount: 5\033[0m"},
		}

		for _, test := range tests {
			var buf bytes.Buffer
			n, err := test.sgr.Fprintf(&buf, test.format, test.args...)
			require.NoError(t, err)
			require.Equal(t, test.expected, buf.String())
			require.Equal(t, len(test.expected), n)
		}
	})
}

func TestSGRs(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		tests := []struct {
			sgrs     SGRs
			expected string
		}{
			{SGRs{FgRed, Bold}, "\033[31;1m"},
			{SGRs{FgGreen, Underline}, "\033[32;4m"},
			{SGRs{BgBlue, FgWhite, Blink}, "\033[44;37;5m"},
			{SGRs{FgLightCyan, BgLightRed}, "\033[96;101m"},
			{SGRs{}, ""},
		}

		for _, test := range tests {
			require.Equal(t, test.expected, test.sgrs.String())
		}
	})

	t.Run("Sprintf", func(t *testing.T) {
		tests := []struct {
			sgrs     SGRs
			format   string
			args     []any
			expected string
		}{
			{SGRs{FgRed, Bold}, "Hello", nil, "\033[31;1mHello\033[0m"},
			{SGRs{FgGreen, Underline}, "Number: %d", []any{42}, "\033[32;4mNumber: 42\033[0m"},
			{SGRs{BgBlue, FgWhite}, "%s: %d", []any{"Count", 5}, "\033[44;37mCount: 5\033[0m"},
			{SGRs{}, "Plain text", nil, "Plain text"},
		}

		for _, test := range tests {
			require.Equal(t, test.expected, test.sgrs.Sprintf(test.format, test.args...))
		}
	})

	t.Run("Fprintf", func(t *testing.T) {
		tests := []struct {
			sgrs     SGRs
			format   string
			args     []any
			expected string
			expLen   int
		}{
			{SGRs{FgRed, Bold}, "Hello", nil, "\033[31;1mHello\033[0m", 16},
			{SGRs{FgGreen, Underline}, "Number: %d", []any{42}, "\033[32;4mNumber: 42\033[0m", 21},
			{SGRs{BgBlue, FgWhite}, "%s: %d", []any{"Count", 5}, "\033[44;37mCount: 5\033[0m", 20},
			{SGRs{}, "Plain text", nil, "Plain text", 10},
		}

		for _, test := range tests {
			var buf bytes.Buffer
			n, err := test.sgrs.Fprintf(&buf, test.format, test.args...)
			require.NoError(t, err)
			require.Equal(t, test.expected, buf.String())
			require.Equal(t, test.expLen, n)
		}
	})
}
