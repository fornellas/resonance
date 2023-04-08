package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/host/agent/api"
	aNet "github.com/fornellas/resonance/host/agent/net"
)

func Ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Pong")
}

func internalServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(500)

	var apiErr api.Error

	if errors.Is(err, fs.ErrPermission) {
		apiErr.Type = "ErrPermission"
	} else if errors.Is(err, fs.ErrNotExist) {
		apiErr.Type = "ErrNotExist"
	} else if _, ok := err.(user.UnknownUserError); ok {
		apiErr.Type = "UnknownUserError"
		apiErr.Message = err.Error()
	} else if _, ok := err.(user.UnknownGroupError); ok {
		apiErr.Type = "UnknownGroupError"
		apiErr.Message = err.Error()
	} else if errors.Is(err, fs.ErrExist) {
		apiErr.Type = "ErrExist"
	} else {
		apiErr.Message = err.Error()
	}

	apiErrBytes, err := yaml.Marshal(&apiErr)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal error: %s", err))
	}

	w.Write(apiErrBytes)
}

func marshalResponse(w http.ResponseWriter, bodyInterface interface{}) {
	body, err := yaml.Marshal(bodyInterface)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal group: %s", err))
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(body)
}

func PostFileFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := mux.Vars(r)["name"]
		if !ok {
			panic("name not found in Vars")
		}
		name = fmt.Sprintf("%c%s", os.PathSeparator, name)

		if !filepath.IsAbs(name) {
			internalServerError(w, fmt.Errorf("must be an absolute path: %s", name))
			return
		}

		decoder := yaml.NewDecoder(r.Body)
		decoder.KnownFields(true)
		var file api.File
		if err := decoder.Decode(&file); err != nil {
			internalServerError(w, fmt.Errorf("fail to unmarshal body: %w", err))
			return
		}

		switch file.Action {
		case api.Chmod:
			if err := os.Chmod(name, file.Mode); err != nil {
				internalServerError(w, err)
			}
		case api.Chown:
			if err := os.Chown(name, file.Uid, file.Gid); err != nil {
				internalServerError(w, err)
			}
		case api.Mkdir:
			if err := os.Mkdir(name, file.Mode); err != nil {
				internalServerError(w, err)
			}
		default:
			internalServerError(w, fmt.Errorf("invalid action: %d", file.Action))
		}
	}
}

func GetLookupFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := mux.Vars(r)["username"]
		if !ok {
			panic("username not found in Vars")
		}

		u, err := user.Lookup(username)
		if err != nil {
			internalServerError(w, err)
			return
		}

		marshalResponse(w, u)
	}
}

func GetLookupGroupFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := mux.Vars(r)["name"]
		if !ok {
			panic("name not found in Vars")
		}

		g, err := user.LookupGroup(name)
		if err != nil {
			internalServerError(w, err)
			return
		}

		marshalResponse(w, g)
	}
}

func GetFileFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := mux.Vars(r)["name"]
		if !ok {
			panic("name not found in Vars")
		}
		name = fmt.Sprintf("%c%s", os.PathSeparator, name)

		if lstat, ok := r.URL.Query()["lstat"]; ok && len(lstat) == 1 && lstat[0] == "true" {
			fileInfo, err := os.Lstat(name)
			if err != nil {
				internalServerError(w, err)
				return
			}
			stat_t := fileInfo.Sys().(*syscall.Stat_t)
			hfi := host.HostFileInfo{
				Name:    filepath.Base(name),
				Size:    fileInfo.Size(),
				Mode:    fileInfo.Mode(),
				ModTime: fileInfo.ModTime(),
				IsDir:   fileInfo.IsDir(),
				Uid:     stat_t.Uid,
				Gid:     stat_t.Gid,
			}
			marshalResponse(w, hfi)
			return
		} else {
			internalServerError(w, errors.New("unknown parameters"))
			return
		}

		// panic("GetFileFn")
	}
}

// func GetMkdirFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Mkdir(ctx context.Context, name string, perm os.FileMode) error {}
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
	router.
		Methods("GET").
		Path("/ping").
		HandlerFunc(Ping)
	router.
		Methods("POST").
		Path("/file/{name:.+}").
		Headers("Content-Type", "application/yaml").
		HandlerFunc(PostFileFn(ctx))
	router.
		Methods("GET").
		Path("/user/{username}").
		HandlerFunc(GetLookupFn(ctx))
	router.
		Methods("GET").
		Path("/group/{name}").
		HandlerFunc(GetLookupGroupFn(ctx))
	router.
		Methods("GET").
		Path("/file/{name:.+}").
		HandlerFunc(GetFileFn(ctx))
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
