package diff

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/kylelemons/godebug/diff"
	"gopkg.in/yaml.v3"
)

type Chunks []diff.Chunk

// HaveChanges return true when the chunks contains changes.
func (cs Chunks) HaveChanges() bool {
	for _, chunk := range cs {
		if len(chunk.Added) > 0 {
			return true
		}
		if len(chunk.Deleted) > 0 {
			return true
		}
	}
	return false
}

func (cs Chunks) added(i int, lines []string, buff *bytes.Buffer) {
	for _, line := range lines {
		if (i == 0 || i == len(cs)-1) && line == "" {
			continue
		}
		if color.NoColor {
			fmt.Fprintf(buff, "+%s\n", line)
		} else {
			reset := color.New(color.Reset)
			reset.Fprintf(buff, "")
			color.New(color.FgGreen).Fprintf(buff, "+%s", line)
			reset.Fprintf(buff, "")
			fmt.Fprintf(buff, "\n")
		}
	}
}

func (cs Chunks) deleted(i int, lines []string, buff *bytes.Buffer) {
	for _, line := range lines {
		if (i == 0 || i == len(cs)-1) && line == "" {
			continue
		}
		if color.NoColor {
			fmt.Fprintf(buff, "-%s\n", line)
		} else {
			reset := color.New(color.Reset)
			reset.Fprintf(buff, "")
			color.New(color.FgRed).Fprintf(buff, "-%s", line)
			reset.Fprintf(buff, "")
			fmt.Fprintf(buff, "\n")
		}
	}
}

func (cs Chunks) equal(i int, lines []string, buff *bytes.Buffer) {
	for _, line := range lines {
		if (i == 0 || i == len(cs)-1) && line == "" {
			continue
		}
		fmt.Fprintf(buff, "%s\n", line)
	}
}

func (cs Chunks) String() string {
	var buff bytes.Buffer
	for i, chunk := range cs {
		cs.added(i, chunk.Added, &buff)
		cs.deleted(i, chunk.Deleted, &buff)
		cs.equal(i, chunk.Equal, &buff)
	}
	return buff.String()
}

// DiffAsYaml converts both interfaces to yaml and diffs them.
func DiffAsYaml(a, b any) Chunks {
	var aStr string
	if a != nil {
		aBytes, err := yaml.Marshal(a)
		if err != nil {
			panic(err)
		}
		aStr = strings.Trim(string(aBytes), "\n")
	}

	var bStr string
	if b != nil {
		bBytes, err := yaml.Marshal(b)
		if err != nil {
			panic(err)
		}
		bStr = strings.Trim(string(bBytes), "\n")
	}

	return diff.DiffChunks(
		strings.Split(aStr, "\n"),
		strings.Split(bStr, "\n"),
	)
}
