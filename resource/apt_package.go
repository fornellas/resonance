package resource

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

// APTPackageParams for APTPackage
type APTPackageParams struct {
	// Package version
	Version string
}

// APTPackage resource manages files.
type APTPackage struct{}

var aptPackageRegexpStatus = regexp.MustCompile(`^Status: (.+)$`)
var aptPackageRegexpVersion = regexp.MustCompile(`^Version: (.+)$`)

func (ap APTPackage) Validate(name Name) error {
	return nil
}

func (ap APTPackage) Check(ctx context.Context, hst host.Host, name Name, parameters yaml.Node) (CheckResult, error) {
	logger := log.GetLogger(ctx)

	// APTPackageParams
	var aptPackageParams APTPackageParams
	if err := parameters.Decode(&aptPackageParams); err != nil {
		return false, err
	}

	// Get package state
	hostCmd := host.Cmd{
		Path: "dpkg",
		Args: []string{"-s", string(name)},
	}
	waitStatus, stdout, stderr, err := hst.Run(ctx, hostCmd)
	if err != nil {
		return false, err
	}
	if !waitStatus.Success() {
		if waitStatus.Exited && waitStatus.ExitCode == 1 && strings.Contains(stderr, "not installed") {
			logger.Debugf("Not installed")
			return false, nil
		}
		return false, fmt.Errorf("failed to call '%s': %s\nstdout:\n%s\nstderr:\n%s", hostCmd.String(), waitStatus.String(), stdout, stderr)
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
		return false, fmt.Errorf("failed to parse state from '%s': %s\nstdout:\n%s\nstderr:\n%s", hostCmd.String(), waitStatus.String(), stdout, stderr)
	}

	// Process
	if status != "install ok installed" {
		logger.Debugf("Status %s", status)
		return false, nil
	}
	if aptPackageParams.Version != "" && aptPackageParams.Version != version {
		logger.Debugf("Version %s", version)
		return false, nil
	}

	return true, nil
}

func (ap APTPackage) Refresh(ctx context.Context, hst host.Host, name Name) error {
	return nil
}

func (ap APTPackage) ConfigureAll(ctx context.Context, hst host.Host, actionDefinitions map[Action]Definitions) error {
	nestedCtx := log.IndentLogger(ctx)

	// Package arguments
	pkgs := []string{}
	for action, definitions := range actionDefinitions {
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
		for name, parameters := range definitions {
			var aptPackageParams APTPackageParams
			if err := parameters.Decode(&aptPackageParams); err != nil {
				return err
			}
			var version string
			if aptPackageParams.Version != "" {
				version = fmt.Sprintf("=%s", aptPackageParams.Version)
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
}
