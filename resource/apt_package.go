package resource

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/fornellas/resonance/host"
)

// APTPackageDesiredState is DesiredState for APTPackage
type APTPackageDesiredState struct {
	// Package version
	Version string
}

func (app *APTPackageDesiredState) Validate() error {
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	if strings.HasSuffix(app.Version, "+") {
		return fmt.Errorf("version can't end in +: %s", app.Version)
	}
	if strings.HasSuffix(app.Version, "-") {
		return fmt.Errorf("version can't end in -: %s", app.Version)
	}
	return nil
}

func (app *APTPackageDesiredState) UnmarshalYAML(node *yaml.Node) error {
	type APTPackageDesiredStateDecode APTPackageDesiredState
	var aptPackageStateDecode APTPackageDesiredStateDecode
	node.KnownFields(true)
	if err := node.Decode(&aptPackageStateDecode); err != nil {
		return err
	}
	aptPackageState := APTPackageDesiredState(aptPackageStateDecode)
	if err := aptPackageState.Validate(); err != nil {
		return fmt.Errorf("line %d: validation error: %w", node.Line, err)
	}
	*app = aptPackageState
	return nil
}

// APTPackageInternalState is InternalState for APTPackage
type APTPackageInternalState struct {
	Status string
}

// APTPackage resource manages files.
type APTPackage struct{}

// var aptPackageRegexpStatus = regexp.MustCompile(`^Status: (.+)$`)
// var aptPackageRegexpVersion = regexp.MustCompile(`^Version: (.+)$`)

func (ap APTPackage) ValidateName(name Name) error {
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#source
	return nil
}

func (ap APTPackage) GetFullState(ctx context.Context, hst host.Host, name Name) (FullState, error) {
	return FullState{}, errors.New("APTPackage.GetFullState")
}

func (ap APTPackage) DiffStates(desired DesiredState, current FullState) []diffmatchpatch.Diff {
	panic(errors.New("APTPackage.CheckState"))
}

func (ap APTPackage) ConfigureAll(ctx context.Context, hst host.Host, actionParameters map[Action]Parameters) error {
	return errors.New("APTPackage.ConfigureAll")
}

func init() {
	MergeableManageableResourcesTypeMap["APTPackage"] = APTPackage{}
	ManageableResourcesDesiredStateMap["APTPackage"] = APTPackageDesiredState{}
	ManageableResourcesInternalStateMap["APTPackage"] = APTPackageInternalState{}
}

// func (ap APTPackage) Check(ctx context.Context, hst host.Host, name Name, state State) (CheckResult, error) {
// 	logger := log.GetLogger(ctx)

// 	// APTPackageDesiredState
// 	aptPackageState := state.(*APTPackageDesiredState)

// 	checkResult := CheckResult(true)

// 	// Get package state
// 	hostCmd := host.Cmd{Path: "dpkg", Args: []string{"-s", string(name)}}
// 	waitStatus, stdout, stderr, err := hst.Run(ctx, hostCmd)
// 	if err != nil {
// 		return false, err
// 	}
// 	if !waitStatus.Success() {
// 		if waitStatus.Exited && waitStatus.ExitCode == 1 && strings.Contains(stderr, "not installed") {
// 			logger.Debugf("Package not installed")
// 			return false, nil
// 		} else {
// 			return false, fmt.Errorf("failed to run '%s': %s\nstdout:\n%s\nstderr:\n%s", hostCmd.String(), waitStatus.String(), stdout, stderr)
// 		}
// 	}

// 	// Parse result
// 	var status, version string
// 	for _, line := range strings.Split(stdout, "\n") {
// 		matches := aptPackageRegexpStatus.FindStringSubmatch(line)
// 		if len(matches) == 2 {
// 			status = matches[1]
// 		}
// 		matches = aptPackageRegexpVersion.FindStringSubmatch(line)
// 		if len(matches) == 2 {
// 			version = matches[1]
// 		}
// 	}
// 	if status == "" || version == "" {
// 		return false, fmt.Errorf("failed to parse state from '%s': %s\nstdout:\n%s\nstderr:\n%s", hostCmd.String(), waitStatus.String(), stdout, stderr)
// 	}

// 	// Process
// 	expectedStatus := "install ok installed"
// 	if status != expectedStatus {
// 		logger.Debugf("Expected status '%s', got '%s'", expectedStatus, status)
// 		checkResult = false
// 	}
// 	if aptPackageState.Version != "" && aptPackageState.Version != version {
// 		logger.Debugf("Expected version %s, got %s", aptPackageState.Version, version)
// 		checkResult = false
// 	}

// 	return checkResult, nil
// }

// func (ap APTPackage) ConfigureAll(
// 	ctx context.Context,
// 	hst host.Host,
// 	actionParameters map[Action]Parameters,
// ) error {
// 	nestedCtx := log.IndentLogger(ctx)

// 	// Package arguments
// 	pkgs := []string{}
// 	for action, parameters := range actionParameters {
// 		var pkgAction string
// 		switch action {
// 		case ActionOk:
// 		case ActionApply:
// 			pkgAction = "+"
// 		case ActionDestroy:
// 			pkgAction = "-"
// 		default:
// 			return fmt.Errorf("unexpected action %s", action)
// 		}
// 		for name, state := range parameters {
// 			aptPackageState := state.(*APTPackageDesiredState)
// 			var version string
// 			if aptPackageState.Version != "" {
// 				version = fmt.Sprintf("=%s", aptPackageState.Version)
// 			}
// 			pkgs = append(pkgs, fmt.Sprintf("%s%s%s", string(name), version, pkgAction))
// 		}
// 	}

// 	// Run apt
// 	cmd := host.Cmd{
// 		Path: "apt-get",
// 		Args: append([]string{"install"}, pkgs...),
// 	}
// 	waitStatus, stdout, stderr, err := hst.Run(nestedCtx, cmd)
// 	if err != nil {
// 		return fmt.Errorf("failed to run '%s': %s", cmd, err)
// 	}
// 	if !waitStatus.Success() {
// 		return fmt.Errorf(
// 			"failed to run '%v': %s:\nstdout:\n%s\nstderr:\n%s",
// 			cmd, waitStatus.String(), stdout, stderr,
// 		)
// 	}

// 	return nil
// }
