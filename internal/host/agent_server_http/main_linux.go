package main

import (
	"bytes"
	"encoding/json"
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

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/internal/host/agent_server_http/api"
	aNet "github.com/fornellas/resonance/internal/host/agent_server_http/net"
	"github.com/fornellas/resonance/internal/host/lib"
)

func marshalResponse(w http.ResponseWriter, bodyInterface interface{}) {
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	err := encoder.Encode(bodyInterface)
	if err != nil {
		panic(fmt.Errorf("failed to encode JSON: %w", err))
	}
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

	marshalResponse(w, &apiErr)
}

func GetuidFn() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		marshalResponse(w, uint64(syscall.Getuid()))
	}
}

func GetgidFn() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		marshalResponse(w, uint64(syscall.Getgid()))
	}
}

func PutFileFn() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := mux.Vars(r)["name"]
		if !ok {
			panic("name not found in Vars")
		}
		name = fmt.Sprintf("%c%s", os.PathSeparator, name)

		modes, ok := r.URL.Query()["mode"]
		if !ok {
			internalServerError(w, errors.New("missing mode from query"))
			return
		}
		if len(modes) != 1 {
			internalServerError(w, fmt.Errorf("received multiple mode: %#v", modes))
			return
		}
		modeInt, err := strconv.ParseUint(modes[0], 10, 32)
		if err != nil {
			internalServerError(w, fmt.Errorf("failed to parse mode: %#v: %s", modes[0], err))
			return
		}
		mode := uint32(modeInt)

		file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fs.FileMode(mode))
		if err != nil {
			internalServerError(w, err)
			return
		}
		defer file.Close()

		if _, err := io.Copy(file, r.Body); err != nil {
			internalServerError(w, err)
			return
		}

		if err := syscall.Chmod(name, mode); err != nil {
			internalServerError(w, err)
			return
		}
	}
}

func GetFileFn() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := mux.Vars(r)["name"]
		if !ok {
			panic("name not found in Vars")
		}
		name = fmt.Sprintf("%c%s", os.PathSeparator, name)

		if lstat, ok := r.URL.Query()["lstat"]; ok && len(lstat) == 1 && lstat[0] == "true" {
			var syscallStat_t syscall.Stat_t
			err := syscall.Lstat(name, &syscallStat_t)
			if err != nil {
				internalServerError(w, err)
				return
			}
			marshalResponse(w, host.Stat_t{
				Dev:     syscallStat_t.Dev,
				Ino:     syscallStat_t.Ino,
				Nlink:   uint64(syscallStat_t.Nlink),
				Mode:    syscallStat_t.Mode,
				Uid:     syscallStat_t.Uid,
				Gid:     syscallStat_t.Gid,
				Rdev:    syscallStat_t.Rdev,
				Size:    syscallStat_t.Size,
				Blksize: int64(syscallStat_t.Blksize),
				Blocks:  syscallStat_t.Blocks,
				Atim: host.Timespec{
					Sec:  int64(syscallStat_t.Atim.Sec),
					Nsec: int64(syscallStat_t.Atim.Nsec),
				},
				Mtim: host.Timespec{
					Sec:  int64(syscallStat_t.Mtim.Sec),
					Nsec: int64(syscallStat_t.Mtim.Nsec),
				},
				Ctim: host.Timespec{
					Sec:  int64(syscallStat_t.Ctim.Sec),
					Nsec: int64(syscallStat_t.Ctim.Nsec),
				},
			})
			return
		} else if readDir, ok := r.URL.Query()["read_dir"]; ok && len(readDir) == 1 && readDir[0] == "true" {
			dirEnts, err := lib.ReadDir(r.Context(), name)
			if err != nil {
				internalServerError(w, err)
				return
			}
			marshalResponse(w, dirEnts)
			return
		} else if readlink, ok := r.URL.Query()["readlink"]; ok && len(readlink) == 1 && readlink[0] == "true" {
			link, err := os.Readlink(name)
			if err != nil {
				internalServerError(w, err)
				return
			}
			marshalResponse(w, link)
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

func PostFileFn() func(http.ResponseWriter, *http.Request) {
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

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var file api.File
		if err := decoder.Decode(&file); err != nil {
			internalServerError(w, fmt.Errorf("fail to decode JSON: %w", err))
			return
		}

		switch file.Action {
		case api.Chmod:
			if err := syscall.Chmod(name, file.Mode); err != nil {
				internalServerError(w, err)
			}
		case api.Chown:
			if err := syscall.Chown(name, int(file.Uid), int(file.Gid)); err != nil {
				internalServerError(w, err)
			}
		case api.Mkdir:
			if err := syscall.Mkdir(name, file.Mode); err != nil {
				internalServerError(w, err)
				return
			}
			if err := syscall.Chmod(name, file.Mode); err != nil {
				internalServerError(w, err)
			}
		case api.Symlink:
			if err := syscall.Symlink(file.Oldname, name); err != nil {
				internalServerError(w, err)
			}
		case api.Mknod:
			if err := syscall.Mknod(name, file.Mode, int(file.Dev)); err != nil {
				internalServerError(w, err)
			}
		default:
			internalServerError(w, fmt.Errorf("invalid action: %d", file.Action))
		}
	}
}

func DeleteFileFn() func(http.ResponseWriter, *http.Request) {
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

func GetGroupFn() func(http.ResponseWriter, *http.Request) {
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

func GetUserFn() func(http.ResponseWriter, *http.Request) {
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

func PostRunFn() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var apiCmd api.Cmd
		if err := decoder.Decode(&apiCmd); err != nil {
			internalServerError(w, fmt.Errorf("fail to decode JSON: %w", err))
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

		waitStatus, err := lib.Run(r.Context(), cmd)
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

func PostShutdownFn() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		go func() {
			time.Sleep(1 * time.Second)
			os.Exit(0)
		}()
	}
}

func main() {
	os.Remove(os.Args[0])

	router := mux.NewRouter()

	router.
		Methods("GET").
		Path("/uid").
		HandlerFunc(GetuidFn())

	router.
		Methods("GET").
		Path("/gid").
		HandlerFunc(GetgidFn())

	router.
		Methods("PUT").
		Path("/file/{name:.*}").
		HandlerFunc(PutFileFn())
	router.
		Methods("GET").
		Path("/file/{name:.*}").
		HandlerFunc(GetFileFn())
	router.
		Methods("POST").
		Path("/file/{name:.*}").
		Headers("Content-Type", "application/json").
		HandlerFunc(PostFileFn())
	router.
		Methods("DELETE").
		Path("/file/{name:.*}").
		HandlerFunc(DeleteFileFn())

	router.
		Methods("GET").
		Path("/group/{name}").
		HandlerFunc(GetGroupFn())

	router.
		Methods("GET").
		Path("/user/{username}").
		HandlerFunc(GetUserFn())

	router.
		Methods("GET").
		Path("/ping").
		HandlerFunc(GetPing)

	router.
		Methods("POST").
		Path("/run").
		Headers("Content-Type", "application/json").
		HandlerFunc(PostRunFn())

	router.
		Methods("POST").
		Path("/shutdown").
		HandlerFunc(PostShutdownFn())

	server := &http2.Server{}

	serveConnOpts := &http2.ServeConnOpts{
		Handler: h2c.NewHandler(router, server),
	}

	ioConn := aNet.IOConn{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}

	server.ServeConn(ioConn, serveConnOpts)
}
