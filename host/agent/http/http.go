package http

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"

	"golang.org/x/net/http2"

	aNet "github.com/fornellas/resonance/host/agent/net"
)

// NewClient creates an http.Client over io.ReadCloser / writer io.WriteCloser
// that only supports the unencrypted "h2c" form of HTTP/2.
func NewClient(reader io.ReadCloser, writer io.WriteCloser) http.Client {
	return http.Client{
		Transport: &http2.Transport{
			DialTLSContext: func(
				ctx context.Context, network, addr string, cfg *tls.Config,
			) (net.Conn, error) {
				return aNet.Conn{
					Reader: reader,
					Writer: writer,
				}, nil
			},
			AllowHTTP: true,
		},
	}
}
