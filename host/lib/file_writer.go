package lib

import (
	"bytes"
	"context"

	"github.com/fornellas/resonance/host/types"
)

// HostFileWriter implements the io.Writer interface for writing to a file on a host.
// It writes content to a file at the given path on the host using the Host's AppendFile method.
type HostFileWriter struct {
	Context context.Context
	Host    types.Host
	Path    string
}

func (wc *HostFileWriter) Write(p []byte) (int, error) {
	if err := wc.Host.AppendFile(wc.Context, wc.Path, bytes.NewBuffer(p), types.FileMode(0600)); err != nil {
		return 0, err
	}
	return len(p), nil
}
