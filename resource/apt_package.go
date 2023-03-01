package resource

import (
	"context"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
)

// APTPackageParams for APTPackage
type APTPackageParams struct {
	// Package version
	Version string
}

// APTPackage resource manages files.
type APTPackage struct{}

func (ap APTPackage) MergeApply() bool {
	return true
}

// var aptPackageRegexpStatus = regexp.MustCompile(`^Status: (.+)$`)
// var aptPackageRegexpVersion = regexp.MustCompile(`^Version: (.+)$`)

func (ap APTPackage) Check(ctx context.Context, hst host.Host, name Name, parameters yaml.Node) (CheckResult, error) {
	// TODO use parameters
	// logger := log.GetLogger(ctx)

	// hostCmd := host.Cmd{
	// 	Path: "dpkg",
	// 	Args: []string{"-s", name.String()},
	// }
	// waitStatus, stdout, stderr, err := hst.Run(ctx, hostCmd)
	// if err != nil {
	// 	return false, err
	// }
	// if !waitStatus.Success() {
	// 	if waitStatus.Exited && waitStatus.ExitCode == 1 && strings.Contains(stderr, "not installed") {
	// 		logger.Debugf("Not installed")
	// 		return false, nil
	// 	}
	// 	return false, fmt.Errorf("failed to call '%s': %s\nstdout:\n%s\nstderr:\n%s", hostCmd.String(), waitStatus.String(), stdout, stderr)
	// }
	// aptPackageState := APTPackageState{}
	// for _, line := range strings.Split(stdout, "\n") {
	// 	matches := aptPackageRegexpStatus.FindStringSubmatch(line)
	// 	if len(matches) == 2 {
	// 		aptPackageState.Status = matches[1]
	// 	}
	// 	matches = aptPackageRegexpVersion.FindStringSubmatch(line)
	// 	if len(matches) == 2 {
	// 		aptPackageState.Version = matches[1]
	// 	}
	// }
	// if aptPackageState.Status == "" || aptPackageState.Version == "" {
	// 	return false, fmt.Errorf("failed to parse state from '%s': %s\nstdout:\n%s\nstderr:\n%s", hostCmd.String(), waitStatus.String(), stdout, stderr)
	// }
	// return true, nil

	return false, errors.New("TODO APTPackage.Check")
}

func (ap APTPackage) Apply(ctx context.Context, hst host.Host, definitions Definitions) error {
	return fmt.Errorf("TODO APTPackage.Apply")
}

func (ap APTPackage) Refresh(ctx context.Context, hst host.Host, name Name) error {
	return nil
}

func (ap APTPackage) Destroy(ctx context.Context, hst host.Host, name Name) error {
	return fmt.Errorf("TODO APTPackage.Destroy")
}

func init() {
	TypeToManageableResource["APTPackage"] = APTPackage{}
}
