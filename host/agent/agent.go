package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host/agent/api"
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

func PostFileFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := mux.Vars(r)["name"]
		if !ok {
			panic("name not found in Vars")
		}
		name = fmt.Sprintf("%c%s", os.PathSeparator, name)

		if !filepath.IsAbs(name) {
			w.WriteHeader(500)
			//  FIXME
			w.Write([]byte(fmt.Sprintf("must be an absolute path: %s", name)))
			return
		}

		decoder := yaml.NewDecoder(r.Body)
		decoder.KnownFields(true)
		var file api.File
		if err := decoder.Decode(&file); err != nil {
			w.WriteHeader(500)
			//  FIXME
			w.Write([]byte(fmt.Sprintf("fail to unmarshal body: %s", err)))
			return
		}

		switch file.Action {
		case api.Chmod:
			if err := os.Chmod(name, file.Mode); err != nil {
				w.WriteHeader(500)

				var apiErr api.Error

				if errors.Is(err, fs.ErrPermission) {
					apiErr.Type = "ErrPermission"
				} else if errors.Is(err, fs.ErrNotExist) {
					apiErr.Type = "ErrNotExist"
				} else {
					apiErr.Message = err.Error()
				}

				apiErrBytes, err := yaml.Marshal(&apiErr)
				if err != nil {
					panic(fmt.Sprintf("failed to marshall error: %s", err))
				}

				w.Write(apiErrBytes)
			}
		default:
			w.WriteHeader(500)
			//  FIXME
			w.Write([]byte(fmt.Sprintf("invalid action: %d", file.Action)))
		}
	}
}

// func GetLookupFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		username := mux.Vars(r)["username"]
// 		u, err := user.Lookup(username)
// 		if err != nil {
// 			w.WriteHeader(500)
// 			w.Write([]byte(fmt.Sprintf("%s", err)))
// 		}
// 		body, err := yaml.Marshal(u)
// 		if err != nil {
// 			w.WriteHeader(500)
// 			w.Write([]byte(fmt.Sprintf("%s", err)))
// 		}
// 		w.Write(body)
// 	}
// }

// func GetLookupGroupFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		name := mux.Vars(r)["name"]
// 		g, err := user.LookupGroup(name)
// 		if err != nil {
// 			w.WriteHeader(500)
// 			w.Write([]byte(fmt.Sprintf("%s", err)))
// 		}
// 		body, err := yaml.Marshal(g)
// 		if err != nil {
// 			w.WriteHeader(500)
// 			w.Write([]byte(fmt.Sprintf("%s", err)))
// 		}
// 		w.Write(body)
// 	}
// }

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

// func GetReadFileFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		name := mux.Vars(r)["name"]
// 		bytes, err := os.ReadFile(name)
// 		if err != nil {
// 			w.WriteHeader(500)
// 			w.Write([]byte(fmt.Sprintf("%s", err)))
// 		}
// 		w.Write(bytes)
// 	}
// }

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
	os.Remove(os.Args[0])

	ctx := context.Background()

	router := mux.NewRouter()
	router.Methods("GET").Path("/ping").HandlerFunc(Ping)
	// router.HandleFunc(Chmod(ctx context.Context, name string, mode os.FileMode) error
	router.
		Methods("POST").
		Path("/file/{name:.+}").
		Headers("Content-Type", "application/yaml").
		HandlerFunc(PostFileFn(ctx))
	// router.Methods("GET").Path("/user/{username}").HandlerFunc(GetLookupFn(ctx))
	// router.Methods("GET").Path("/group/{name}").HandlerFunc(GetLookupGroupFn(ctx))
	// router.Methods("TBD").Path("/tbd").HandlerFunc(GetLstatFn(ctx))
	// router.Methods("TBD").Path("/tbd").HandlerFunc(GetMkdirFn(ctx))
	// router.Methods("GET").Path("/file/{name:.+}").HandlerFunc(GetReadFileFn(ctx))
	// router.Methods("TBD").Path("/tbd").HandlerFunc(GetRemoveFn(ctx))
	// router.Methods("TBD").Path("/tbd").HandlerFunc(GetRunFn(ctx))
	// router.Methods("TBD").Path("/tbd").HandlerFunc(GetWriteFileFn(ctx))

	server := &http2.Server{}

	serveConnOpts := &http2.ServeConnOpts{
		Handler: h2c.NewHandler(router, server),
	}

	conn := aNet.Conn{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}

	server.ServeConn(conn, serveConnOpts)
}
