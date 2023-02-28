package resource

import (
	"context"
	"fmt"

	"github.com/fornellas/resonance/host"
)

// APTPackageParams for APTPackage
type APTPackageParams struct {
	// Package version
	Version string
}

// APTPackageState for APTPackage
type APTPackageState struct {
	Version string
	Status  string
}

// APTPackage resource manages files.
type APTPackage struct{}

func (ap APTPackage) MergeApply() bool {
	return true
}

// func (ap APTPackage) GetDesiredState(parameters yaml.Node) (State, error) {
// 	aptPackageParams := APTPackageParams{}
// 	if err := parameters.Decode(&aptPackageParams); err != nil {
// 		return nil, err
// 	}
// 	return APTPackageState{
// 		Version: aptPackageParams.Version,
// 		Status:  "install ok installed",
// 	}, nil
// }

// var aptPackageRegexpStatus = regexp.MustCompile(`^Status: (.+)$`)
// var aptPackageRegexpVersion = regexp.MustCompile(`^Version: (.+)$`)

// func (ap APTPackage) GetState(ctx context.Context, hst host.Host, name Name) (State, error) {
// 	hostCmd := host.Cmd{
// 		Path: "dpkg",
// 		Args: []string{"-s", name.String()},
// 	}
// 	waitStatus, stdout, stderr, err := hst.Run(ctx, hostCmd)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if !waitStatus.Success() {
// 		if waitStatus.Exited && waitStatus.ExitCode == 1 && strings.Contains(stderr, "not installed") {
// 			return APTPackageState{
// 				Status: "not installed",
// 			}, nil
// 		}
// 		return nil, fmt.Errorf("failed to call '%s': %s\nstdout:\n%s\nstderr:\n%s", hostCmd.String(), waitStatus.String(), stdout, stderr)
// 	}
// 	aptPackageState := APTPackageState{}
// 	for _, line := range strings.Split(stdout, "\n") {
// 		matches := aptPackageRegexpStatus.FindStringSubmatch(line)
// 		if len(matches) == 2 {
// 			aptPackageState.Status = matches[1]
// 		}
// 		matches = aptPackageRegexpVersion.FindStringSubmatch(line)
// 		if len(matches) == 2 {
// 			aptPackageState.Version = matches[1]
// 		}
// 	}
// 	if aptPackageState.Status == "" || aptPackageState.Version == "" {
// 		return nil, fmt.Errorf("failed to parse state from '%s': %s\nstdout:\n%s\nstderr:\n%s", hostCmd.String(), waitStatus.String(), stdout, stderr)
// 	}
// 	return aptPackageState, nil
// }

func (ap APTPackage) Apply(ctx context.Context, hst host.Host, instances []Instance) error {
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
