package resource

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

// APTPackageStateParameters is StateParameters for APTPackage
type APTPackageStateParameters struct {
	// Package version
	Version string `yaml:"version"`
}

func (apsp APTPackageStateParameters) Validate() error {
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	if strings.HasSuffix(apsp.Version, "+") {
		return fmt.Errorf("version can't end in +: %s", apsp.Version)
	}
	if strings.HasSuffix(apsp.Version, "-") {
		return fmt.Errorf("version can't end in -: %s", apsp.Version)
	}
	return nil
}

// APTPackageInternalState is InternalState for APTPackage
type APTPackageInternalState struct {
	Status string
}

// APTPackage resource manages files.
type APTPackage struct{}

var aptPackageRegexpStatus = regexp.MustCompile(`^Status: (.+)$`)
var aptPackageRegexpVersion = regexp.MustCompile(`^Version: (.+)$`)

func (ap APTPackage) ValidateName(name Name) error {
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#source
	return nil
}

func (ap APTPackage) GetFullState(ctx context.Context, hst host.Host, name Name) (*FullState, error) {
	logger := log.GetLogger(ctx)

	stateParameters := APTPackageStateParameters{}
	internalState := APTPackageInternalState{}

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
	stateParameters.Version = version
	internalState.Status = status

	return &FullState{
		StateParameters: &stateParameters,
		InternalState:   &internalState,
	}, nil
}

func (ap APTPackage) DiffStates(
	ctx context.Context, hst host.Host,
	desiredStateParameters StateParameters, currentFullState FullState,
) ([]diffmatchpatch.Diff, error) {
	diffs := []diffmatchpatch.Diff{}
	desiredAPTPackageStateParameters := desiredStateParameters.(*APTPackageStateParameters)
	currentAPTPackageStateParameters := currentFullState.StateParameters.(*APTPackageStateParameters)
	currentAPTPackageInternalState := currentFullState.InternalState.(*APTPackageInternalState)

	diffs = append(diffs, Diff(currentAPTPackageStateParameters, desiredAPTPackageStateParameters)...)
	diffs = append(diffs, Diff(currentAPTPackageInternalState, APTPackageInternalState{
		Status: "install ok installed",
	})...)

	return diffs, nil
}

func (ap APTPackage) ConfigureAll(
	ctx context.Context, hst host.Host, actionParameters map[Action]Parameters,
) error {
	nestedCtx := log.IndentLogger(ctx)

	// Package arguments
	pkgs := []string{}
	for action, parameters := range actionParameters {
		var pkgAction string
		switch action {
		case ActionOk:
		case ActionApply:
			pkgAction = "+"
		case ActionDestroy:
			pkgAction = "-"
		default:
			return fmt.Errorf("unexpected action %s", action)
		}
		for name, stateParameters := range parameters {
			var version string
			if stateParameters != nil {
				aptPackageState := stateParameters.(*APTPackageStateParameters)
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
	MergeableManageableResourcesTypeMap["APTPackage"] = APTPackage{}
	ManageableResourcesStateParametersMap["APTPackage"] = APTPackageStateParameters{}
	ManageableResourcesInternalStateMap["APTPackage"] = APTPackageInternalState{}
}
