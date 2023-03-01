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

func (ap APTPackage) ConfigureAll(ctx context.Context, hst host.Host, actionDefinition map[Action]Definitions) error {
	return fmt.Errorf("TODO APTPackage.Apply")
}

func init() {
	MergeableManageableResourcesTypeMap["APTPackage"] = APTPackage{}
}
