package resource

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

// SystemdUnitParams for SystemdUnit.
// It has the same attributes as FileParams, but if Content is empty,
// then it is assumed the unit file was added by another resource (eg:
// a package) and it gonna be left as is.
type SystemdUnitParams FileParams

// SystemdUnit resource manages Systemd Units.
// These units are enabled and reload-or-restart on apply or refresh.
type SystemdUnit struct{}

func (su SystemdUnit) Validate(name Name) error {
	path := string(name)
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}
	return nil
}

func checkUnitFile(
	ctx context.Context,
	hst host.Host,
	name Name,
	parameters Parameters,
	path,
	job string,
	systemdUnitParams *SystemdUnitParams,
) (CheckResult, error) {
	// Pre-existing unit file
	if systemdUnitParams.Content == "" {
		// It must exist
		_, err := hst.Lstat(ctx, path)
		if err != nil {
			if os.IsNotExist(err) {
				return false, fmt.Errorf("unit file not found: it must either exist or have its contents defined")
			}
			return false, err
		}
		// It must be known to systemctl
		cmd := host.Cmd{Path: "systemctl", Args: []string{"list-unit-files", job}}
		waitStatus, stdout, stderr, err := hst.Run(ctx, cmd)
		if err != nil {
			return false, fmt.Errorf("failed to run '%s': %s", cmd, err)
		}
		if !waitStatus.Success() {
			return false, fmt.Errorf(
				"unit file not found by systemctl: it must either exist or have its contents defined\n'%v': %s:\nstdout:\n%s\nstderr:\n%s",
				cmd, waitStatus.String(), stdout, stderr,
			)
		}
		// Must create unit file
	} else {
		file := IndividuallyManageableResourceTypeMap["File"]
		checkResult, err := file.Check(ctx, hst, name, (*FileParams)(systemdUnitParams))
		if err != nil {
			return false, err
		}
		if !checkResult {
			return checkResult, nil
		}
	}
	return true, nil
}

func getSystemdUnitProperties(ctx context.Context, hst host.Host, job string) (map[string]string, error) {
	cmd := host.Cmd{Path: "systemctl", Args: []string{"show", job}}
	waitStatus, stdout, stderr, err := hst.Run(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run '%s': %s", cmd, err)
	}
	if !waitStatus.Success() {
		return nil, fmt.Errorf(
			"failed to run '%v': %s:\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	properties := map[string]string{}
	for _, line := range strings.Split(stdout, "\n") {
		if line == "" {
			continue
		}
		tokens := strings.Split(line, "=")
		if len(tokens) != 2 {
			return nil, fmt.Errorf("unexpected output from '%s', expected 'key=value', got '%s'", cmd, line)
		}
		properties[tokens[0]] = tokens[1]
	}
	return properties, nil
}

func (su SystemdUnit) Check(ctx context.Context, hst host.Host, name Name, parameters Parameters) (CheckResult, error) {
	logger := log.GetLogger(ctx)

	path := string(name)
	job := filepath.Base(path)

	// SystemdUnitParams
	systemdUnitParams := parameters.(*SystemdUnitParams)

	checkResult, err := checkUnitFile(ctx, hst, name, parameters, path, job, systemdUnitParams)
	if err != nil {
		return false, err
	}
	if !checkResult {
		return false, err
	}

	// Get job properties
	properties, err := getSystemdUnitProperties(ctx, hst, job)
	if err != nil {
		return false, err
	}

	// Check enabled
	// https://github.com/systemd/systemd/blob/master/src/systemctl/systemctl-is-enabled.c
	unitFileState, ok := properties["UnitFileState"]
	if !ok {
		return false, errors.New("property UnitFileStatenot found for job")
	}
	switch unitFileState {
	case "enabled":
	case "static":
	case "generated":
	case "alias":
	case "indirect":
	default:
		logger.Debug("not enabled")
		return false, nil
	}

	// Check active
	activeState, ok := properties["ActiveState"]
	if !ok {
		return false, errors.New("property ActiveState found for job")
	}
	if activeState != "active" {
		logger.Debug("not active")
		return false, nil
	}

	return true, nil
}

func (su SystemdUnit) Apply(ctx context.Context, hst host.Host, name Name, parameters Parameters) error {
	return errors.New("TODO SystemdUnit.Apply")
}

func (su SystemdUnit) Refresh(ctx context.Context, hst host.Host, name Name) error {
	return errors.New("TODO SystemdUnit.Refresh")
}

func (su SystemdUnit) Destroy(ctx context.Context, hst host.Host, name Name) error {
	return errors.New("TODO SystemdUnit.Destroy")
}

func init() {
	IndividuallyManageableResourceTypeMap["SystemdUnit"] = SystemdUnit{}
	ManageableResourcesParametersMap["SystemdUnit"] = SystemdUnitParams{}
}
