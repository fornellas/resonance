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

func (app *APTPackageParams) Validate() error {
	if strings.HasSuffix(app.Version, "+") {
		return fmt.Errorf("version can't end in +: %s", app.Version)
	}
	if strings.HasSuffix(app.Version, "-") {
		return fmt.Errorf("version can't end in -: %s", app.Version)
	}
	return nil
}

func (app *APTPackageParams) UnmarshalYAML(node *yaml.Node) error {
	type APTPackageParamsDecode APTPackageParams
	var aptPackageParamsDecode APTPackageParamsDecode
	node.KnownFields(true)
	if err := node.Decode(&aptPackageParamsDecode); err != nil {
		return err
	}
	aptPackageParams := APTPackageParams(aptPackageParamsDecode)
	if err := aptPackageParams.Validate(); err != nil {
		return fmt.Errorf("line %d: validation error: %w", node.Line, err)
	}
	*app = aptPackageParams
	return nil
}

// APTPackage resource manages files.
type APTPackage struct{}

var aptPackageRegexpStatus = regexp.MustCompile(`^Status: (.+)$`)
var aptPackageRegexpVersion = regexp.MustCompile(`^Version: (.+)$`)

func (ap APTPackage) Validate(name Name) error {
	return nil
}

func (ap APTPackage) Check(ctx context.Context, hst host.Host, name Name, parameters Parameters) (CheckResult, error) {
	logger := log.GetLogger(ctx)

	// APTPackageParams
	aptPackageParams := parameters.(*APTPackageParams)

	checkResult := CheckResult(true)

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
			logger.Debugf("Package not installed")
			return false, nil
		} else {
			return false, fmt.Errorf("failed to run '%s': %s\nstdout:\n%s\nstderr:\n%s", hostCmd.String(), waitStatus.String(), stdout, stderr)
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
		return false, fmt.Errorf("failed to parse state from '%s': %s\nstdout:\n%s\nstderr:\n%s", hostCmd.String(), waitStatus.String(), stdout, stderr)
	}

	// Process
	expectedStatus := "install ok installed"
	if status != expectedStatus {
		logger.Debugf("Expected status '%s', got '%s'", expectedStatus, status)
		checkResult = false
	}
	if aptPackageParams.Version != "" && aptPackageParams.Version != version {
		logger.Debugf("Expected version %s, got %s", aptPackageParams.Version, version)
		checkResult = false
	}

	return checkResult, nil
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
			aptPackageParams := parameters.(*APTPackageParams)
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
	ManageableResourcesParametersMap["APTPackage"] = APTPackageParams{}
}
