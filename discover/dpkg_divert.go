package discover

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

type DpkgDivert struct {
	Filename string
	// DivertTo is the location where the versions of file, as provided by other packages, will be
	// diverted.
	DivertTo string
	// Package  is the name of a package whose copy of file will not be diverted.
	// i.e. file will be diverted for all packages except package.
	//
	// If unset, it means local, specifies that all packages' versions of this file are
	// diverted.  This means, that there are no exceptions, and whatever package is installed,
	// the file is diverted. This can be used by an admin to install a locally modified version.
	Package *string
}

type DpkgDiverts struct {
	dpkgDiverts             []*DpkgDivert
	filenameToDpkgDivertMap map[string]*DpkgDivert
}

func (d *DpkgDiverts) GetDpkgDivert(filename string) *DpkgDivert {
	if dpkgDivert, ok := d.filenameToDpkgDivertMap[filename]; ok {
		return dpkgDivert
	}
	return nil
}

var dpkgDivertLocalRegexp = regexp.MustCompile(`^local diversion of (.+) to (.+)$`)
var dpkgDivertPackageRegexp = regexp.MustCompile(`^diversion of (.+) to (.+) by (.+)$`)

func parseDpkgDivert(stdoutReader io.Reader) (*DpkgDiverts, error) {
	scanner := bufio.NewScanner(stdoutReader)

	dpkgDiverts := &DpkgDiverts{
		filenameToDpkgDivertMap: map[string]*DpkgDivert{},
	}

	for scanner.Scan() {
		line := scanner.Text()

		dpkgDivert := &DpkgDivert{}
		dpkgDiverts.dpkgDiverts = append(dpkgDiverts.dpkgDiverts, dpkgDivert)

		match := false

		matches := dpkgDivertLocalRegexp.FindStringSubmatch(line)
		if len(matches) == 3 {
			dpkgDivert.Filename = matches[1]
			dpkgDivert.DivertTo = matches[2]
			match = true
		}

		matches = dpkgDivertPackageRegexp.FindStringSubmatch(line)
		if len(matches) == 4 {
			dpkgDivert.Filename = matches[1]
			dpkgDivert.DivertTo = matches[2]
			dpkgDivert.Package = &matches[3]
			match = true
		}

		if !match {
			return nil, fmt.Errorf("failed to parse line: %#v", line)
		}
		dpkgDiverts.filenameToDpkgDivertMap[dpkgDivert.Filename] = dpkgDivert
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}
	return dpkgDiverts, nil
}

func LoadDpkgDiverts(ctx context.Context, host types.Host) (*DpkgDiverts, error) {
	ctx, _ = log.MustContextLoggerSection(ctx, "diversions")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stdoutReader, stdoutWritter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stderrBuffer := bytes.Buffer{}

	cmd := types.Cmd{
		Path:   "/usr/bin/dpkg-divert",
		Args:   []string{"--list"},
		Stdout: stdoutWritter,
		Stderr: &stderrBuffer,
	}

	errCh := make(chan error)
	go func() {
		var runErr error
		defer func() {
			errCh <- stdoutWritter.Close()
			errCh <- runErr
			close(errCh)
		}()

		var waitStatus types.WaitStatus
		waitStatus, runErr = host.Run(ctx, cmd)
		if !waitStatus.Success() {
			errCh <- fmt.Errorf("%s:\nstderr:\n%s", cmd, stderrBuffer.String())
			return
		}
	}()

	dpkgDiverts, err := parseDpkgDivert(stdoutReader)
	if err != nil {
		return nil, err
	}

	err = nil
	for chErr := range errCh {
		err = errors.Join(err, chErr)
	}

	return dpkgDiverts, err

}
