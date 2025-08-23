package resources

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
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

// parseAlternativeStanza parses an alternative stanza starting at index i.
func (a *DpkgAlternative) parseAlternativeStanza(lines []string, i int) (AlternativeChoice, int) {
	choice := AlternativeChoice{
		Alternative: strings.TrimSpace(strings.SplitN(lines[i], ":", 2)[1]),
		Slaves:      map[string]string{},
	}
	j := i + 1
	for ; j < len(lines); j++ {
		if lines[j] == "" {
			continue
		}
		if strings.HasPrefix(lines[j], "Alternative:") {
			break
		}
		if strings.HasPrefix(lines[j], "Priority:") {
			priority, _ := strconv.Atoi(strings.TrimSpace(strings.SplitN(lines[j], ":", 2)[1]))
			choice.Priority = priority
		} else if strings.HasPrefix(lines[j], "Slaves:") {
			for k := j + 1; k < len(lines); k++ {
				if lines[k] == "" || (len(lines[k]) > 0 && lines[k][0] != ' ') {
					j = k - 1
					break
				}
				parts := strings.Fields(strings.TrimSpace(lines[k]))
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
		switch {
		case strings.HasPrefix(line, "Name:"):
			a.Name = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		case strings.HasPrefix(line, "Link:"):
			a.Link = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		case strings.HasPrefix(line, "Status:"):
			a.Status = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		case strings.HasPrefix(line, "Value:"):
			a.Value = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		case strings.HasPrefix(line, "Alternative:"):
			choice, nextIdx := a.parseAlternativeStanza(lines, i)
			choices = append(choices, choice)
			i = nextIdx
		}
		i++
	}
	a.Choices = choices
	return nil
}
