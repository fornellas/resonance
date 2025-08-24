package resources

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fornellas/resonance/host/lib"
	"github.com/fornellas/resonance/host/types"
)

// AlternativeChoice describe available alternatives for a link group.
type AlternativeChoice struct {
	// Path to this alternative.
	Alternative string
	// Value of the priority of this alternative.
	Priority int
	// All slave alternatives associated to the master link of the alternative.
	// This is a map from the generic name of the slave alternative to the path to the slave alternative.
	Slaves map[string]string
}

// DpkgAlternative manages the state of dpkg alternatives via update-alternatives(1).
type DpkgAlternative struct {
	// The alternative name in the alternative directory.
	Name string
	// Absent is true when no alternatives exist.
	Absent bool
	// The generic name of the alternative.
	Link string
	// Slave links associated to the master link of the alternative.
	// This is a map from the generic name of the slave alternative to the path to the slave link.
	Slaves map[string]string
	// The status of the alternative (auto or manual).
	Status string
	// The path of the currently selected alternative. It can also take the magic value none. It is
	// used if the link doesn't exist.
	Value string
	// Available alternatives in the link group.
	Choices []AlternativeChoice
}

// Validate whether AlternativeChoice is valid.
func (c *AlternativeChoice) Validate() error {
	var errs []error
	if c.Alternative == "" {
		errs = append(errs, fmt.Errorf("alternative path is empty"))
	} else if !filepath.IsAbs(c.Alternative) {
		errs = append(errs, fmt.Errorf("alternative path is not absolute: %s", c.Alternative))
	} else if filepath.Clean(c.Alternative) != c.Alternative {
		errs = append(errs, fmt.Errorf("alternative path is not clean: %s", c.Alternative))
	}
	for slaveName, slavePath := range c.Slaves {
		if slaveName == "" {
			errs = append(errs, fmt.Errorf("slave name is empty"))
		}
		if slavePath == "" {
			errs = append(errs, fmt.Errorf("slave path for %s is empty", slaveName))
		} else if !filepath.IsAbs(slavePath) {
			errs = append(errs, fmt.Errorf("slave path for %s is not absolute: %s", slaveName, slavePath))
		} else if filepath.Clean(slavePath) != slavePath {
			errs = append(errs, fmt.Errorf("slave path for %s is not clean: %s", slaveName, slavePath))
		}
	}
	return errors.Join(errs...)
}

// Validate whether DpkgAlternative is valid.
func (a *DpkgAlternative) Validate() error {
	var errs []error
	if a.Name == "" {
		errs = append(errs, fmt.Errorf("Name is empty"))
	}
	if a.Absent {
		// All fields except Name must be zero values
		if a.Link != "" {
			errs = append(errs, fmt.Errorf("Link must be empty when Absent is true"))
		}
		if len(a.Slaves) != 0 {
			errs = append(errs, fmt.Errorf("Slaves must be empty when Absent is true"))
		}
		if a.Status != "" {
			errs = append(errs, fmt.Errorf("Status must be empty when Absent is true"))
		}
		if a.Value != "" {
			errs = append(errs, fmt.Errorf("Value must be empty when Absent is true"))
		}
		if len(a.Choices) != 0 {
			errs = append(errs, fmt.Errorf("Choices must be empty when Absent is true"))
		}
		return errors.Join(errs...)
	}
	if a.Link == "" {
		errs = append(errs, fmt.Errorf("Link is empty"))
	} else if !filepath.IsAbs(a.Link) {
		errs = append(errs, fmt.Errorf("Link is not absolute: %s", a.Link))
	} else if filepath.Clean(a.Link) != a.Link {
		errs = append(errs, fmt.Errorf("Link is not clean: %s", a.Link))
	}
	for slaveName, slavePath := range a.Slaves {
		if slaveName == "" {
			errs = append(errs, fmt.Errorf("slave name is empty"))
		}
		if slavePath == "" {
			errs = append(errs, fmt.Errorf("slave path for %s is empty", slaveName))
		} else if !filepath.IsAbs(slavePath) {
			errs = append(errs, fmt.Errorf("slave path for %s is not absolute: %s", slaveName, slavePath))
		} else if filepath.Clean(slavePath) != slavePath {
			errs = append(errs, fmt.Errorf("slave path for %s is not clean: %s", slaveName, slavePath))
		}
	}
	if a.Status != "auto" && a.Status != "manual" {
		errs = append(errs, fmt.Errorf("invalid Status: %s", a.Status))
	}
	// For manual, Value must be set and valid
	if a.Status == "manual" {
		if a.Value == "" {
			errs = append(errs, fmt.Errorf("Value must be set when Status is manual"))
		} else if a.Value != "none" {
			if !filepath.IsAbs(a.Value) {
				errs = append(errs, fmt.Errorf("Value is not absolute: %s", a.Value))
			} else if filepath.Clean(a.Value) != a.Value {
				errs = append(errs, fmt.Errorf("Value is not clean: %s", a.Value))
			}
		}
	} else if a.Status == "auto" {
		if a.Value != "" {
			errs = append(errs, fmt.Errorf("Value must be empty when Status is auto"))
		}
	}
	for i, choice := range a.Choices {
		if err := choice.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("Choices[%d]: %w", i, err))
		}
	}
	// Ensure every slave generic name in Slaves exists in at least one Choices[i].Slaves
	for slaveName := range a.Slaves {
		found := false
		for _, choice := range a.Choices {
			if _, ok := choice.Slaves[slaveName]; ok {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Errorf("slave %q in Slaves is missing from all Choices", slaveName))
		}
	}
	return errors.Join(errs...)
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
		// Handle missing alternative case
		if waitStatus.ExitCode == 2 && stderr == "update-alternatives: error: no alternatives for "+a.Name+"\n" {
			*a = DpkgAlternative{
				Name:   a.Name,
				Absent: true,
			}
			return nil
		}
		return fmt.Errorf("%s failed: %s\n\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), stdout, stderr)
	}

	var choices []AlternativeChoice
	a.Slaves = nil
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
			if a.Status == "auto" {
				a.Value = ""
			}
		case strings.HasPrefix(line, "Slaves:"):
			for j := i + 1; j < len(lines); j++ {
				if lines[j] == "" {
					continue
				}
				if !strings.HasPrefix(lines[j], " ") {
					i = j - 1
					break
				}
				parts := strings.Fields(strings.TrimSpace(lines[j]))
				if len(parts) == 2 {
					if a.Slaves == nil {
						a.Slaves = map[string]string{}
					}
					a.Slaves[parts[0]] = parts[1]
				}
			}
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

// Add new alternatives or update priorities/slaves.
func (a *DpkgAlternative) addNewAlternatives(ctx context.Context, host types.Host) error {
	for _, choice := range a.Choices {
		args := []string{
			"--install",
			a.Link,
			a.Name,
			choice.Alternative,
			strconv.Itoa(choice.Priority),
		}
		for genericName, path := range choice.Slaves {
			link := a.Slaves[genericName]
			args = append(args, "--slave", link, genericName, path)
		}
		cmd := types.Cmd{
			Path: "update-alternatives",
			Args: args,
		}
		waitStatus, stdout, stderr, err := lib.Run(ctx, host, cmd)
		if err != nil {
			return err
		}
		if !waitStatus.Success() {
			return fmt.Errorf("%s failed: %s\n\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), stdout, stderr)
		}
	}
	return nil
}

// Set selected alternative if needed.
func (a *DpkgAlternative) setSelectedAlternative(ctx context.Context, host types.Host) error {
	// Only set if manual and Value is set and changed
	if a.Status == "manual" {
		cmd := types.Cmd{
			Path: "update-alternatives",
			Args: []string{"--set", a.Name, a.Value},
		}
		waitStatus, stdout, stderr, err := lib.Run(ctx, host, cmd)
		if err != nil {
			return err
		}
		if !waitStatus.Success() {
			return fmt.Errorf("%s failed: %s\n\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), stdout, stderr)
		}
	}
	return nil
}

// Set mode if needed.
func (a *DpkgAlternative) setMode(ctx context.Context, host types.Host) error {
	if a.Status == "auto" {
		cmd := types.Cmd{
			Path: "update-alternatives",
			Args: []string{"--auto", a.Name},
		}
		waitStatus, stdout, stderr, err := lib.Run(ctx, host, cmd)
		if err != nil {
			return err
		}
		if !waitStatus.Success() {
			return fmt.Errorf("%s failed: %s\n\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), stdout, stderr)
		}
		// Clear Value when switching to auto (for Apply logic)
		a.Value = ""
	}
	return nil
}

// Apply the state to host.
func (a *DpkgAlternative) Apply(ctx context.Context, host types.Host) error {
	cmd := types.Cmd{
		Path: "update-alternatives",
		Args: []string{"--remove-all", a.Name},
	}
	waitStatus, stdout, stderr, err := lib.Run(ctx, host, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		if !(waitStatus.ExitCode == 2 && stderr == "update-alternatives: error: no alternatives for "+a.Name+"\n") {
			return fmt.Errorf("%s failed: %s\n\nSTDOUT:\n%s\nSTDERR:\n%s", cmd, waitStatus.String(), stdout, stderr)
		}
	}

	if a.Absent {
		return nil
	}

	current := &DpkgAlternative{Name: a.Name}
	if err := current.Load(ctx, host); err != nil {
		return err
	}

	if err := a.addNewAlternatives(ctx, host); err != nil {
		return err
	}

	if err := a.setSelectedAlternative(ctx, host); err != nil {
		return err
	}

	if err := a.setMode(ctx, host); err != nil {
		return err
	}

	return nil
}
