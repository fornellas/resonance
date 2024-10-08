package net

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"go.uber.org/multierr"
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

// Conn implements net.Conn for io.ReadCloser / io.WriteCloser.
type Conn struct {
	Reader io.ReadCloser
	Writer io.WriteCloser
}

func (c Conn) Read(b []byte) (int, error) {
	n, err := c.Reader.Read(b)
	return n, err
}

func (c Conn) Write(b []byte) (int, error) {
	n, err := c.Writer.Write(b)
	return n, err
}

func (c Conn) Close() error {
	return multierr.Combine(
		c.Reader.Close(),
		c.Writer.Close(),
	)
}

func (c Conn) LocalAddr() net.Addr {
	return Addr{
		Reader: c.Reader,
	}
}

func (c Conn) RemoteAddr() net.Addr {
	return Addr{
		Writer: c.Writer,
	}
}

func (c Conn) SetDeadline(t time.Time) error {
	return os.ErrNoDeadline
}

func (c Conn) SetReadDeadline(t time.Time) error {
	return os.ErrNoDeadline
}

func (c Conn) SetWriteDeadline(t time.Time) error {
	return os.ErrNoDeadline
}

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
	return Addr{
		Reader: nil,
		Writer: nil,
	}
}
