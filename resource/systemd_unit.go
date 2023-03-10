package resource

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/fornellas/resonance/host"
)

// SystemdUnitStateParameters for SystemdUnit.
// It has the same attributes as FileStateParameters, but if Content is empty,
// then it is assumed the unit file was added by another resource (eg:
// a package) and it gonna be left as is.
type SystemdUnitStateParameters FileStateParameters

func (susp SystemdUnitStateParameters) Validate() error {
	return nil
}

// SystemdUnitInternalState is InternalState for SystemdUnit
type SystemdUnitInternalState struct {
}

// SystemdUnit resource manages Systemd Units.
// These units are enabled and reload-or-restart on apply or refresh.
type SystemdUnit struct{}

func (su SystemdUnit) ValidateName(name Name) error {
	path := string(name)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}
	return nil
}

func (su SystemdUnit) GetFullState(ctx context.Context, hst host.Host, name Name) (FullState, error) {
	return FullState{}, errors.New("SystemdUnit.GetFullState")
}

func (su SystemdUnit) DiffStates(
	ctx context.Context, hst host.Host,
	desiredStateParameters StateParameters, currentFullState FullState,
) ([]diffmatchpatch.Diff, error) {
	panic(errors.New("SystemdUnit.DiffStates"))
}

func (su SystemdUnit) Refresh(ctx context.Context, hst host.Host, name Name) error {
	return errors.New("SystemdUnit.Refresh")
}

func (su SystemdUnit) ConfigureAll(
	ctx context.Context, hst host.Host, actionParameters map[Action]Parameters,
) error {
	return errors.New("SystemdUnit.ConfigureAll")
}

func init() {
	MergeableManageableResourcesTypeMap["SystemdUnit"] = SystemdUnit{}
	ManageableResourcesStateParametersMap["SystemdUnit"] = SystemdUnitStateParameters{}
	ManageableResourcesInternalStateMap["SystemdUnit"] = SystemdUnitInternalState{}
}

// func checkUnitFile(
// 	ctx context.Context,
// 	hst host.Host,
// 	name Name,
// 	state State,
// 	path,
// 	job string,
// 	systemdUnitState *SystemdUnitStateParameters,
// ) (CheckResult, error) {
// 	// Pre-existing unit file
// 	if systemdUnitState.Content == "" {
// 		// It must exist
// 		_, err := hst.Lstat(ctx, path)
// 		if err != nil {
// 			if os.IsNotExist(err) {
// 				return false, fmt.Errorf("unit file not found: it must either exist or have its contents defined")
// 			}
// 			return false, err
// 		}
// 		// It must be known to systemctl
// 		cmd := host.Cmd{Path: "systemctl", Args: []string{"list-unit-files", job}}
// 		waitStatus, stdout, stderr, err := hst.Run(ctx, cmd)
// 		if err != nil {
// 			return false, fmt.Errorf("failed to run '%s': %s", cmd, err)
// 		}
// 		if !waitStatus.Success() {
// 			return false, fmt.Errorf(
// 				"unit file not found by systemctl: it must either exist or have its contents defined\n'%v': %s:\nstdout:\n%s\nstderr:\n%s",
// 				cmd, waitStatus.String(), stdout, stderr,
// 			)
// 		}
// 	} else {
// 		// Must create unit file
// 		file := IndividuallyManageableResourceTypeMap["File"]
// 		checkResult, err := file.Check(ctx, hst, name, (*FileState)(systemdUnitState))
// 		if err != nil {
// 			return false, err
// 		}
// 		if !checkResult {
// 			return checkResult, nil
// 		}
// 	}
// 	return true, nil
// }

// func getSystemdUnitProperties(ctx context.Context, hst host.Host, job string) (map[string]string, error) {
// 	cmd := host.Cmd{Path: "systemctl", Args: []string{"show", job}}
// 	waitStatus, stdout, stderr, err := hst.Run(ctx, cmd)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to run '%s': %s", cmd, err)
// 	}
// 	if !waitStatus.Success() {
// 		return nil, fmt.Errorf(
// 			"failed to run '%v': %s:\nstdout:\n%s\nstderr:\n%s",
// 			cmd, waitStatus.String(), stdout, stderr,
// 		)
// 	}
// 	properties := map[string]string{}
// 	for _, line := range strings.Split(stdout, "\n") {
// 		if line == "" {
// 			continue
// 		}
// 		tokens := strings.Split(line, "=")
// 		if len(tokens) != 2 {
// 			return nil, fmt.Errorf("unexpected output from '%s', expected 'key=value', got '%s'", cmd, line)
// 		}
// 		properties[tokens[0]] = tokens[1]
// 	}
// 	return properties, nil
// }

// func (su SystemdUnit) Check(ctx context.Context, hst host.Host, name Name, state State) (CheckResult, error) {
// 	logger := log.GetLogger(ctx)

// 	path := string(name)
// 	job := filepath.Base(path)

// 	// SystemdUnitStateParameters
// 	systemdUnitState := state.(*SystemdUnitStateParameters)

// 	// Unit file
// 	checkResult, err := checkUnitFile(ctx, hst, name, state, path, job, systemdUnitState)
// 	if err != nil {
// 		return false, err
// 	}
// 	if !checkResult {
// 		return false, err
// 	}

// 	// Get job properties
// 	properties, err := getSystemdUnitProperties(ctx, hst, job)
// 	if err != nil {
// 		return false, err
// 	}

// 	// Check enabled
// 	// https://github.com/systemd/systemd/blob/master/src/systemctl/systemctl-is-enabled.c
// 	unitFileState, ok := properties["UnitFileState"]
// 	if !ok {
// 		return false, errors.New("property UnitFileStatenot found for job")
// 	}
// 	switch unitFileState {
// 	case "enabled":
// 	case "static":
// 	case "generated":
// 	case "alias":
// 	case "indirect":
// 	default:
// 		logger.Debug("not enabled")
// 		return false, nil
// 	}

// 	// Check active
// 	activeState, ok := properties["ActiveState"]
// 	if !ok {
// 		return false, errors.New("property ActiveState found for job")
// 	}
// 	if activeState != "active" {
// 		logger.Debug("not active")
// 		return false, nil
// 	}

// 	return true, nil
// }

// func (su SystemdUnit) Apply(ctx context.Context, hst host.Host, name Name, state State) error {
// 	// logger := log.GetLogger(ctx)

// 	// path := string(name)
// 	// job := filepath.Base(path)

// 	// // SystemdUnitStateParameters
// 	// systemdUnitState := state.(*SystemdUnitStateParameters)

// 	// // Create unit file
// 	// if systemdUnitState.Content != "" {
// 	// 	file := IndividuallyManageableResourceTypeMap["File"]
// 	// 	err := file.Apply(ctx, hst, name, (*FileState)(systemdUnitState))
// 	// 	if err != nil {
// 	// 		return false, err
// 	// 	}
// 	// }

// 	// Enable
// 	// systemctl enable job

// 	// Activate

// 	// systemctl daemon-reload

// 	return errors.New("TODO SystemdUnit.Apply")
// }

// func (su SystemdUnit) Refresh(ctx context.Context, hst host.Host, name Name) error {
// 	// systemctl reload-or-restart $unit
// 	return errors.New("TODO SystemdUnit.Refresh")
// }

// func (su SystemdUnit) Destroy(ctx context.Context, hst host.Host, name Name) error {

// 	// systemctl daemon-reload
// 	return errors.New("TODO SystemdUnit.Destroy")
// }
