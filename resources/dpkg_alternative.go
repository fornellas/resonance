package resources

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
)

// AlternativeChoice describe available alternatives for a link group.
type AlternativeChoice struct {
	// Path to this stanza's alternative.
	Alternative string
	// Value of the priority of this alternative.
	Priority int
	// Slave alternatives associated to the master link of the alternative.
	// It maps the generic name of the slave alternative to the path to the slave alternative.
	Slaves map[string]string
}

// DpkgAlternative manages the state of dpkg alternatives via update-alternatives(1).
type DpkgAlternative struct {
	// The alternative name in the alternative directory.
	Name string
	// The generic name of the alternative.
	Link string
	// The status of the alternative (auto or manual).
	Status string
	// The path of the currently selected alternative. It can also take the magic value none. It is
	// used if the link doesn't exist.
	Value string
	// Available alternatives in the link group.
	Choices []AlternativeChoice
}

func (a *DpkgAlternative) hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func (a *DpkgAlternative) trimField(s string) string {
	for i := range s {
		if s[i] == ':' {
			return a.trimSpace(s[i+1:])
		}
	}
	return a.trimSpace(s)
}

func (a *DpkgAlternative) trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func (a *DpkgAlternative) splitFields(s string) []string {
	var fields []string
	start := 0
	for i := range s {
		if s[i] == ' ' {
			fields = append(fields, s[start:i])
			start = i + 1
			for start < len(s) && s[start] == ' ' {
				start++
			}
			break
		}
	}
	if start < len(s) {
		fields = append(fields, s[start:])
	}
	return fields
}

func (a *DpkgAlternative) parseInt(s string) int {
	n := 0
	for i := range s {
		if s[i] < '0' || s[i] > '9' {
			break
		}
		n = n*10 + int(s[i]-'0')
	}
	return n
}

// parseAlternativeStanza parses an alternative stanza starting at index i.
func (a *DpkgAlternative) parseAlternativeStanza(lines []string, i int) (AlternativeChoice, int) {
	choice := AlternativeChoice{
		Alternative: a.trimField(lines[i]),
		Slaves:      map[string]string{},
	}
	j := i + 1
	for ; j < len(lines); j++ {
		if lines[j] == "" {
			continue
		}
		if a.hasPrefix(lines[j], "Alternative:") {
			break
		}
		if a.hasPrefix(lines[j], "Priority:") {
			choice.Priority = a.parseInt(a.trimField(lines[j]))
		} else if a.hasPrefix(lines[j], "Slaves:") {
			for k := j + 1; k < len(lines); k++ {
				if lines[k] == "" || (len(lines[k]) > 0 && lines[k][0] != ' ') {
					j = k - 1
					break
				}
				parts := a.splitFields(a.trimSpace(lines[k]))
				if len(parts) == 2 {
					choice.Slaves[parts[0]] = parts[1]
				}
			}
		}
	}
	return choice, j - 1
}

// Loads the full state of the alternative from host (Name must be set).
func (a *DpkgAlternative) Load(ctx context.Context, host types.Host) error {
	cmd := types.Cmd{
		Path: "update-alternatives",
		Args: []string{"--query", a.Name},
	}
	waitStatus, stdout, stderr, err := lib.Run(ctx, host, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		return fmt.Errorf("%s failed: %s\n\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), stdout, stderr)
	}

	var choices []AlternativeChoice
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	for i := 0; i < len(lines); {
		line := lines[i]
		if line == "" {
			i++
			continue
		}
		if a.hasPrefix(line, "Name:") {
			a.Name = a.trimField(line)
			i++
			continue
		}
		if a.hasPrefix(line, "Link:") {
			a.Link = a.trimField(line)
			i++
			continue
		}
		if a.hasPrefix(line, "Status:") {
			a.Status = a.trimField(line)
			i++
			continue
		}
		if a.hasPrefix(line, "Value:") {
			a.Value = a.trimField(line)
			i++
			continue
		}
		if a.hasPrefix(line, "Alternative:") {
			choice, nextIdx := a.parseAlternativeStanza(lines, i)
			choices = append(choices, choice)
			i = nextIdx + 1
			continue
		}
		i++
	}
	a.Choices = choices
	return nil
}
