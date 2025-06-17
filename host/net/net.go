package net

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// Addr implements net.Addr for io.ReadCloser / io.WriteCloser.
type Addr struct {
	Reader io.ReadCloser
	Writer io.WriteCloser
}

func (a Addr) Network() string {
	return "io"
}

func (a Addr) String() string {
	return fmt.Sprintf("%v<>%v", a.Reader, a.Writer)
}

// IOConn implements net.IOConn for io.ReadCloser / io.WriteCloser.
type IOConn struct {
	Reader io.ReadCloser
	Writer io.WriteCloser
}

func (c IOConn) Read(b []byte) (int, error) {
	return c.Reader.Read(b)
}

func (c IOConn) Write(b []byte) (int, error) {
	return c.Writer.Write(b)
}

func (c IOConn) Close() error {
	return errors.Join(
		c.Reader.Close(),
		c.Writer.Close(),
	)
}

func (c IOConn) LocalAddr() net.Addr {
	return Addr{
		Reader: c.Reader,
	}
}

func (c IOConn) RemoteAddr() net.Addr {
	return Addr{
		Writer: c.Writer,
	}
}

func (c IOConn) SetDeadline(t time.Time) error {
	return os.ErrNoDeadline
}

func (c IOConn) SetReadDeadline(t time.Time) error {
	return os.ErrNoDeadline
}

func (c IOConn) SetWriteDeadline(t time.Time) error {
	return os.ErrNoDeadline
}
