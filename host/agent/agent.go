package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/user"

	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gopkg.in/yaml.v3"

	aNet "github.com/fornellas/resonance/host/agent/net"
)

func Ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Pong")
}

// func GetChmodFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Chmod(ctx context.Context, name string, mode os.FileMode) error {}
// 	}
// }

// func GetChownFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Chown(ctx context.Context, name string, uid, gid int) error {}
// 	}
// }

func GetLookupFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		u, err := user.Lookup(username)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("%s", err)))
		}
		body, err := yaml.Marshal(u)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("%s", err)))
		}
		w.Write(body)
	}
}

func GetLookupGroupFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["name"]
		g, err := user.LookupGroup(name)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("%s", err)))
		}
		body, err := yaml.Marshal(g)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("%s", err)))
		}
		w.Write(body)
	}
}

// func GetLookupGroupFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// LookupGroup(ctx context.Context, name string) (*user.Group, error) {}
// 	}
// }

// func GetLstatFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Lstat(ctx context.Context, name string) (HostFileInfo, error) {}
// 	}
// }

// func GetMkdirFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Mkdir(ctx context.Context, name string, perm os.FileMode) error {}
// 	}
// }

func GetReadFileFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["name"]
		bytes, err := os.ReadFile(name)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("%s", err)))
		}
		w.Write(bytes)
	}
}

// func GetRemoveFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Remove(ctx context.Context, name string) error {}
// 	}
// }

// func GetRunFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {}
// 	}
// }

// func GetWriteFileFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {}
// 	}
// }

func main() {

	// handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprint(os.Stderr, "GOT\n")
	// 	// time.Sleep(1 * time.Second)
	// 	fmt.Fprint(w, "Hello world")
	// })

	ctx := context.Background()

	router := mux.NewRouter()
	router.Methods("GET").Path("/ping").HandlerFunc(Ping)
	// router.HandleFunc(Chmod(ctx context.Context, name string, mode os.FileMode) error
	// router.HandleFunc(Chown(ctx context.Context, name string, uid, gid int) error
	router.Methods("GET").Path("/user/{username}").HandlerFunc(GetLookupFn(ctx))
	router.Methods("GET").Path("/group/{name}").HandlerFunc(GetLookupGroupFn(ctx))
	// router.Methods("TBD").Path("/tbd").HandlerFunc(GetLstatFn(ctx))
	// router.Methods("TBD").Path("/tbd").HandlerFunc(GetMkdirFn(ctx))
	router.Methods("GET").Path("/file/{name:.+}").HandlerFunc(GetReadFileFn(ctx))
	// router.Methods("TBD").Path("/tbd").HandlerFunc(GetRemoveFn(ctx))
	// router.Methods("TBD").Path("/tbd").HandlerFunc(GetRunFn(ctx))
	// router.Methods("TBD").Path("/tbd").HandlerFunc(GetWriteFileFn(ctx))

	server := &http2.Server{
		MaxHandlers: 1,
	}

	serveConnOpts := &http2.ServeConnOpts{
		Handler: h2c.NewHandler(router, server),
	}

	conn := aNet.Conn{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}

	defer func() {
		os.Remove(os.Args[0])
	}()

	server.ServeConn(conn, serveConnOpts)
}
