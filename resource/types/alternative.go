package resource

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/resource"
)

// AltertativeState for Altertatives
type AltertativeState struct {
	// The status of the alternative (auto or manual).
	Status string `yaml:"status"`
	// Alternative Path
	Path string `yaml:"path"`
}

func (as AltertativeState) ValidateAndUpdate(ctx context.Context, hst host.Host) (resource.State, error) {
	if as.Status == "auto" {
		return nil, fmt.Errorf("can not set 'path' with auto 'status'")
	} else if as.Status == "manual" {
		if !filepath.IsAbs(as.Path) {
			return nil, fmt.Errorf("path must be absolute: %#v", as.Path)
		}
	} else {
		return nil, fmt.Errorf("status must be either 'auto' or 'manual'")
	}

	return as, nil
}

// Altertatives manages the Debian altertatives system (update-alternatives)
// https://wiki.debian.org/DebianAlternatives
type Altertatives struct{}

func (a Altertatives) ValidateName(name resource.Name) error {
	return nil
}

func (a Altertatives) GetState(ctx context.Context, hst host.Host, name resource.Name) (resource.State, error) {
	alternativeState := AltertativeState{}

	cmd := types.Cmd{
		Path: "update-alternatives",
		Args: []string{
			"--query", string(name),
		},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, hst, cmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		return nil, fmt.Errorf(
			"failed to run '%s': %s\nstdout:\n%s\nstderr:\n%s",
			cmd.String(), waitStatus.String(), stdout, stderr,
		)
	}

	info := map[string]string{}
	for _, line := range strings.Split(stdout, "\n") {
		if len(line) == 0 {
			break
		}

		tokens := strings.Split(line, ": ")
		if len(tokens) != 2 {
			return nil, fmt.Errorf(
				"failed to parse output: %s: expected 'foo: bar' format: %#v",
				cmd, line,
			)
		}
		info[tokens[0]] = tokens[1]
	}

	var ok bool

	alternativeState.Status, ok = info["Status"]
	if !ok {
		return nil, fmt.Errorf("'Status' missing from output: %s:\n%s", cmd, stdout)
	}

	if alternativeState.Status == "manual" {
		alternativeState.Path, ok = info["Value"]
		if !ok {
			return nil, fmt.Errorf("'Value' missing from output: %s:\n%s", cmd, stdout)
		}
	}

	return alternativeState.ValidateAndUpdate(ctx, hst)
}

func (a Altertatives) Configure(
	ctx context.Context, hst host.Host, name resource.Name, state resource.State,
) error {
	alternativeState := state.(AltertativeState)

	cmd := types.Cmd{
		Path: "update-alternatives",
	}
	if alternativeState.Status == "auto" {
		cmd.Args = []string{"--auto", string(name)}
	} else if alternativeState.Status == "manual" {
		cmd.Args = []string{"--set", string(name), alternativeState.Path}
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, hst, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		return fmt.Errorf(
			"failed to run '%s': %s\nstdout:\n%s\nstderr:\n%s",
			cmd.String(), waitStatus.String(), stdout, stderr,
		)
	}

	return nil
}

func (a Altertatives) Destroy(ctx context.Context, hst host.Host, name resource.Name) error {
	// TODO restore to previous state
	return nil
}

func init() {
	resource.IndividuallyManageableResourceTypeMap["Altertatives"] = Altertatives{}
	resource.ManageableResourcesStateMap["Altertatives"] = AltertativeState{}
}
