package proc

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/fornellas/resonance/host/types"
)

type Mount struct {
	Device     string
	MountPoint string
	FSType     string
	Options    string
	Dump       int
	Pass       int
}

type Mounts []Mount

func LoadMounts(ctx context.Context, host types.Host) (Mounts, error) {
	procMountsReadCloser, err := host.ReadFile(ctx, "/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer procMountsReadCloser.Close()

	var mounts Mounts
	scanner := bufio.NewScanner(procMountsReadCloser)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 6 {
			return nil, fmt.Errorf("/proc/mounts: invalid line: %s", line)
		}

		var dump, pass int
		fmt.Sscanf(fields[4], "%d", &dump)
		fmt.Sscanf(fields[5], "%d", &pass)

		mount := Mount{
			Device:     fields[0],
			MountPoint: fields[1],
			FSType:     fields[2],
			Options:    fields[3],
			Dump:       dump,
			Pass:       pass,
		}

		mounts = append(mounts, mount)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return mounts, nil
}
