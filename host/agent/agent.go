package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
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
			hfi := types.HostFileInfo{
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
			contexts, err := os.ReadFile(name)
			if err != nil {
				internalServerError(w, err)
				return
			}
			w.Write(contexts)
		}
	}
}

func DeleteRemoveFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := mux.Vars(r)["name"]
		if !ok {
			panic("name not found in Vars")
		}
		name = fmt.Sprintf("%c%s", os.PathSeparator, name)

		if err := os.Remove(name); err != nil {
			internalServerError(w, err)
		}
	}
}

func PostRunFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := yaml.NewDecoder(r.Body)
		decoder.KnownFields(true)
		var apiCmd api.Cmd
		if err := decoder.Decode(&apiCmd); err != nil {
			internalServerError(w, fmt.Errorf("fail to unmarshal body: %w", err))
			return
		}

		var stdin io.Reader
		if apiCmd.Stdin != nil {
			stdin = bytes.NewReader(apiCmd.Stdin)
		}

		var stdout []byte
		var stdoutBuff *bytes.Buffer
		if apiCmd.Stdout {
			stdoutBuff = bytes.NewBuffer(stdout)
		}

		var stderr []byte
		var stderrBuff *bytes.Buffer
		if apiCmd.Stderr {
			stderrBuff = bytes.NewBuffer(stderr)
		}

		cmd := types.Cmd{
			Path:   apiCmd.Path,
			Args:   apiCmd.Args,
			Env:    apiCmd.Env,
			Dir:    apiCmd.Dir,
			Stdin:  stdin,
			Stdout: stdoutBuff,
			Stderr: stderrBuff,
		}

		waitStatus, err := host.LocalRun(ctx, cmd)
		if err != nil {
			internalServerError(w, err)
			return
		}

		cmdResponse := api.CmdResponse{
			WaitStatus: waitStatus,
		}

		if stdoutBuff != nil {
			cmdResponse.Stdout = stdoutBuff.Bytes()
		}

		if stderrBuff != nil {
			cmdResponse.Stderr = stderrBuff.Bytes()
		}

		marshalResponse(w, cmdResponse)
	}
}

// func GetWriteFileFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {}
// 	}
// }

func main() {
	os.Remove(os.Args[0])

	ctx := context.Background()
	ctx = log.SetLoggerValue(ctx, os.Stderr, "error", func(code int) { os.Exit(code) })

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
	router.
		Methods("DELETE").
		Path("/file/{name:.+}").
		HandlerFunc(DeleteRemoveFn(ctx))
	router.
		Methods("POST").
		Path("/run").
		Headers("Content-Type", "application/yaml").
		HandlerFunc(PostRunFn(ctx))
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
