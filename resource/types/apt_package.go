package resource

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"
)

// APTPackageState is State for APTPackage
type APTPackageState struct {
	// Package version
	Version string `yaml:"version"`
}

func (aps APTPackageState) Validate() error {
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	if strings.HasSuffix(aps.Version, "+") {
		return fmt.Errorf("version can't end in +: %s", aps.Version)
	}
	if strings.HasSuffix(aps.Version, "-") {
		return fmt.Errorf("version can't end in -: %s", aps.Version)
	}
	return nil
}

// APTPackage resource manages files.
type APTPackage struct{}

var aptPackageRegexpStatus = regexp.MustCompile(`^Status: (.+)$`)
var aptPackageRegexpVersion = regexp.MustCompile(`^Version: (.+)$`)

func (ap APTPackage) ValidateName(name resource.Name) error {
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#source
	return nil
}

func (ap APTPackage) GetState(ctx context.Context, hst host.Host, name resource.Name) (resource.State, error) {
	logger := log.GetLogger(ctx)

	// Get package state
	hostCmd := host.Cmd{Path: "dpkg", Args: []string{"-s", string(name)}}
	waitStatus, stdout, stderr, err := hst.Run(ctx, hostCmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		if waitStatus.Exited && waitStatus.ExitCode == 1 && strings.Contains(stderr, "not installed") {
			logger.Debug("Not installed")
			return nil, nil
		} else {
			return nil, fmt.Errorf("failed to run '%s': %s\nstdout:\n%s\nstderr:\n%s", hostCmd.String(), waitStatus.String(), stdout, stderr)
		}
	}

	// Parse result
	var status, version string
	for _, line := range strings.Split(stdout, "\n") {
		matches := aptPackageRegexpStatus.FindStringSubmatch(line)
		if len(matches) == 2 {
			status = matches[1]
		}
		matches = aptPackageRegexpVersion.FindStringSubmatch(line)
		if len(matches) == 2 {
			version = matches[1]
		}
	}
	if status == "" || version == "" {
		return nil, fmt.Errorf("failed to parse state from '%s': %s\nstdout:\n%s\nstderr:\n%s", hostCmd.String(), waitStatus.String(), stdout, stderr)
	}

	return APTPackageState{
		Version: version,
	}, nil
}

func (ap APTPackage) DiffStates(
	ctx context.Context, hst host.Host,
	desiredState resource.State, currentState resource.State,
) ([]diffmatchpatch.Diff, error) {
	diffs := []diffmatchpatch.Diff{}
	desiredAPTPackageState := desiredState.(*APTPackageState)
	currentAPTPackageState := currentState.(*APTPackageState)

	diffs = append(diffs, resource.Diff(currentAPTPackageState, desiredAPTPackageState)...)

	return diffs, nil
}

func (ap APTPackage) ConfigureAll(
	ctx context.Context, hst host.Host, actionNameStateMap map[resource.Action]map[resource.Name]resource.State,
) error {
	nestedCtx := log.IndentLogger(ctx)

	// Package arguments
	pkgs := []string{}
	for action, nameStateMap := range actionNameStateMap {
		var pkgAction string
		switch action {
		case resource.ActionOk:
		case resource.ActionApply:
			pkgAction = "+"
		case resource.ActionDestroy:
			pkgAction = "-"
		default:
			return fmt.Errorf("unexpected action %s", action)
		}
		for name, state := range nameStateMap {
			var version string
			if state != nil {
				aptPackageState := state.(*APTPackageState)
				if aptPackageState.Version != "" {
					version = fmt.Sprintf("=%s", aptPackageState.Version)
				}
			}
			pkgs = append(pkgs, fmt.Sprintf("%s%s%s", string(name), version, pkgAction))
		}
	}

	// Run apt
	cmd := host.Cmd{
		Path: "apt-get",
		Args: append([]string{"install"}, pkgs...),
	}
	waitStatus, stdout, stderr, err := hst.Run(nestedCtx, cmd)
	if err != nil {
		return fmt.Errorf("failed to run '%s': %s", cmd, err)
	}
	if !waitStatus.Success() {
		return fmt.Errorf(
			"failed to run '%v': %s:\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}

	return nil
}

func init() {
	resource.MergeableManageableResourcesTypeMap["APTPackage"] = APTPackage{}
	resource.ManageableResourcesStateMap["APTPackage"] = APTPackageState{}
}
