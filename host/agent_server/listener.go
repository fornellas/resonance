package main

import (
	"io"
	"net"

	hostNet "github.com/fornellas/resonance/host/net"
)

// Listener implements net.Listener for a single connection.
//
// It accepts a single connection and then returns an EOF error on subsequent calls to Accept().
// This is useful for wrapping an existing connection into a listener interface.
type Listener struct {
	connChan chan net.Conn
}

func NewListener(conn net.Conn) *Listener {
	l := &Listener{
		connChan: make(chan net.Conn, 1),
	}
	l.connChan <- conn
	return l
}

func (l *Listener) Accept() (net.Conn, error) {
	conn, ok := <-l.connChan
	if !ok {
		return nil, io.EOF
	}
	return conn, nil
}

func (l *Listener) Close() error {
	close(l.connChan)
	return nil
}

func (l *Listener) Addr() net.Addr {
	return hostNet.Addr{
		Reader: nil,
		Writer: nil,
	}
}
