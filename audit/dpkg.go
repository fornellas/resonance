package audit

import (
	"bufio"
	"context"
	"fmt"
	"path/filepath"

	"strings"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

// TODO dpkg-divert --list
type DpkgPackage struct {
	BinaryPackage string
	Version       string
	DbFsysFiles   []string
	SourcePackage string
	Conffiles     []string
}

type DpkgDb struct {
	DpkgPackages          []*DpkgPackage
	PathToDpkgPackagesMap map[string][]*DpkgPackage
}

//gocyclo:ignore
func NewDpkgDb(ctx context.Context, host types.Host) (*DpkgDb, error) {
	ctx, _ = log.MustContextLoggerSection(ctx, "Listing all dpkg packages")

	dpkgWcmd := types.Cmd{
		Path: "/usr/bin/dpkg-query",
		Args: []string{
			"--show",
			"--showformat=" + strings.Join([]string{
				"binary:Package=${binary:Package}",
				"Version=${Version}",
				"db-fsys:Files=", "${db-fsys:Files}",
				"source:Package=${source:Package}",
				"Conffiles=", "${Conffiles}",
				"end\n",
			}, "\n"),
		},
	}
	waitStatus, stdout, _, err := types.Run(ctx, host, dpkgWcmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		return nil, fmt.Errorf("%s:%s", dpkgWcmd, waitStatus.String())
	}

	scanner := bufio.NewScanner(strings.NewReader(stdout))

	dpkgDb := &DpkgDb{
		DpkgPackages:          []*DpkgPackage{},
		PathToDpkgPackagesMap: map[string][]*DpkgPackage{},
	}
	dpkgPackage := &DpkgPackage{}
	var stringSliceValue *[]string = nil
	var valuer func(string) (string, error) = nil
	for scanner.Scan() {
		line := scanner.Text()

		if line == "end" {
			for _, path := range dpkgPackage.DbFsysFiles {
				dpkgDb.PathToDpkgPackagesMap[path] = append(dpkgDb.PathToDpkgPackagesMap[path], dpkgPackage)
			}
			for _, path := range dpkgPackage.Conffiles {
				dpkgDb.PathToDpkgPackagesMap[path] = append(dpkgDb.PathToDpkgPackagesMap[path], dpkgPackage)
			}
			dpkgDb.DpkgPackages = append(dpkgDb.DpkgPackages, dpkgPackage)
			dpkgPackage = &DpkgPackage{}
			stringSliceValue = nil
			continue
		}

		if stringSliceValue == nil || (!strings.HasPrefix(line, " ") && len(line) != 0) {
			index := strings.Index(line, "=")
			if index == -1 {
				return nil, fmt.Errorf("unexpected line: %#v", line)
			}
			key := line[:index]
			value := line[index+1:]
			stringSliceValue = nil
			switch key {
			case "binary:Package":
				dpkgPackage.BinaryPackage = value
			case "Version":
				dpkgPackage.Version = value
			case "db-fsys:Files":
				dpkgPackage.DbFsysFiles = []string{}
				stringSliceValue = &dpkgPackage.DbFsysFiles
				valuer = func(line string) (string, error) {
					return filepath.Clean(line), nil
				}
			case "source:Package":
				dpkgPackage.SourcePackage = value
			case "Conffiles":
				dpkgPackage.Conffiles = []string{}
				stringSliceValue = &dpkgPackage.Conffiles
				valuer = func(line string) (string, error) {
					lastSpaceIndex := strings.LastIndex(line, " ")
					if lastSpaceIndex == -1 {
						return "", fmt.Errorf("bug: invalid Conffile: %#v", line)
					}
					return filepath.Clean(line[:lastSpaceIndex]), nil
				}
			default:
				return nil, fmt.Errorf("unexpected key: %#v", key)
			}
		} else {
			if len(line) == 0 {
				continue
			}
			value, err := valuer(line[1:])
			if err != nil {
				return nil, err
			}
			*stringSliceValue = append(*stringSliceValue, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return dpkgDb, nil
}

// func (d *DpkgDb) Verify() (bool, string, error) {
// 	// TODO dpkg --verify
// 	// TODO md5sums
// 	return true, "TODO", errors.New("TODO")
// }
