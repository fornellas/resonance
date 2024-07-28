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
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	iHost "github.com/fornellas/resonance/internal/host"
	"github.com/fornellas/resonance/internal/host/agent/api"
	aNet "github.com/fornellas/resonance/internal/host/agent/net"
)

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

func PutFileFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := mux.Vars(r)["name"]
		if !ok {
			panic("name not found in Vars")
		}
		name = fmt.Sprintf("%c%s", os.PathSeparator, name)

		perms, ok := r.URL.Query()["perm"]
		if !ok {
			internalServerError(w, errors.New("missing perm from query"))
			return
		}
		if len(perms) != 1 {
			internalServerError(w, fmt.Errorf("received multiple perm: %#v", perms))
			return
		}
		permInt, err := strconv.ParseInt(perms[0], 10, 32)
		if err != nil {
			internalServerError(w, fmt.Errorf("failed to parse perm: %#v: %s", perms[0], err))
			return
		}
		perm := os.FileMode(permInt)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			internalServerError(w, err)
			return
		}

		if err := os.WriteFile(name, body, perm); err != nil {
			internalServerError(w, err)
			return
		}
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
			contexts, err := os.ReadFile(name)
			if err != nil {
				internalServerError(w, err)
				return
			}
			w.Write(contexts)
		}
	}
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

func DeleteFileFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
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

func GetGroupFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
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

func GetPing(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Pong")
}

func GetUserFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
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

		cmd := host.Cmd{
			Path:   apiCmd.Path,
			Args:   apiCmd.Args,
			Env:    apiCmd.Env,
			Dir:    apiCmd.Dir,
			Stdin:  stdin,
			Stdout: stdoutBuff,
			Stderr: stderrBuff,
		}

		waitStatus, err := iHost.LocalRun(ctx, cmd)
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

func PostShutdownFn(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		go func() {
			time.Sleep(1 * time.Second)
			os.Exit(0)
		}()
	}
}

func main() {
	os.Remove(os.Args[0])

	ctx := context.Background()

	router := mux.NewRouter()

	router.
		Methods("PUT").
		Path("/file/{name:.+}").
		HandlerFunc(PutFileFn(ctx))
	router.
		Methods("GET").
		Path("/file/{name:.+}").
		HandlerFunc(GetFileFn(ctx))
	router.
		Methods("POST").
		Path("/file/{name:.+}").
		Headers("Content-Type", "application/yaml").
		HandlerFunc(PostFileFn(ctx))
	router.
		Methods("DELETE").
		Path("/file/{name:.+}").
		HandlerFunc(DeleteFileFn(ctx))

	router.
		Methods("GET").
		Path("/group/{name}").
		HandlerFunc(GetGroupFn(ctx))

	router.
		Methods("GET").
		Path("/user/{username}").
		HandlerFunc(GetUserFn(ctx))

	router.
		Methods("GET").
		Path("/ping").
		HandlerFunc(GetPing)

	router.
		Methods("POST").
		Path("/run").
		Headers("Content-Type", "application/yaml").
		HandlerFunc(PostRunFn(ctx))

	router.
		Methods("POST").
		Path("/shutdown").
		HandlerFunc(PostShutdownFn(ctx))

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
