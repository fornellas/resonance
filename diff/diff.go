// Diff related utilities.
package diff

import (
	"bytes"
	"fmt"

	"github.com/fatih/color"
	"github.com/fornellas/slogxt/ansi"
	"github.com/kylelemons/godebug/diff"
)

// Chunks represents a collection of chunks that describe the difference between two
// texts. Each chunk describes a series of added, deleted, and equal lines.
// The primary purpose of Chunks is to display text differences in a readable format,
// with added lines prefixed with '+' and deleted lines prefixed with '-'.
// ANSI colors are used unless color.NoColor is set.
type Chunks []diff.Chunk

// HasChanges return true when the chunks contains changes.
func (c Chunks) HasChanges() bool {
	for _, chunk := range c {
		if len(chunk.Added) > 0 {
			return true
		}
		if len(chunk.Deleted) > 0 {
			return true
		}
	}
	return false
}

func (c Chunks) added(i int, lines []string, buff *bytes.Buffer) {
	for _, line := range lines {
		if (i == 0 || i == len(c)-1) && line == "" {
			continue
		}
		ansi.FgGreen.Fprintf(buff, "+%s", line)
		fmt.Fprintf(buff, "\n")
	}
}

func (c Chunks) deleted(i int, lines []string, buff *bytes.Buffer) {
	for _, line := range lines {
		if (i == 0 || i == len(c)-1) && line == "" {
			continue
		}
		reset := color.New(color.Reset)
		reset.Fprintf(buff, "")
		color.New(color.FgRed).Fprintf(buff, "-%s", line)
		reset.Fprintf(buff, "")
		fmt.Fprintf(buff, "\n")
	}
}

func (c Chunks) equal(i int, lines []string, buff *bytes.Buffer) {
	for _, line := range lines {
		if (i == 0 || i == len(c)-1) && line == "" {
			continue
		}
		fmt.Fprintf(buff, "%s\n", line)
	}
}

func (c Chunks) TerminalString() string {
	var buff bytes.Buffer
	for i, chunk := range c {
		c.added(i, chunk.Added, &buff)
		c.deleted(i, chunk.Deleted, &buff)
		c.equal(i, chunk.Equal, &buff)
	}
	return buff.String()
}
