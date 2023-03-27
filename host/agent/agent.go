package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	aNet "github.com/fornellas/resonance/host/agent/net"
)

func main() {
	server := &http2.Server{
		MaxHandlers: 1,
	}

	conn := aNet.Conn{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(os.Stderr, "GOT\n")
		time.Sleep(1 * time.Second)
		fmt.Fprint(w, "Hello world")
	})

	serveConnOpts := &http2.ServeConnOpts{
		Handler: h2c.NewHandler(handler, server),
	}

	server.ServeConn(conn, serveConnOpts)
}
