package host

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"go.uber.org/multierr"
	"golang.org/x/net/http2"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/internal/host/agent_server_http/api"
	aNet "github.com/fornellas/resonance/internal/host/agent_server_http/net"

	"github.com/fornellas/resonance/log"
)

var AgentHttpBinGz = map[string][]byte{}

// AgentHttpClient interacts with a given Host using an agent that's copied and ran at the
// host.
type AgentHttpClient struct {
	Host       host.Host
	path       string
	Client     *http.Client
	spawnErrCh chan error
}

func getHttpAgentBinGz(ctx context.Context, hst host.Host) ([]byte, error) {
	cmd := host.Cmd{
		Path: "uname",
		Args: []string{"-m"},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, hst, cmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		return nil, fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	goarch, err := getGoArch(strings.TrimRight(stdout, "\n"))
	if err != nil {
		return nil, err
	}
	osArch := fmt.Sprintf("linux.%s", goarch)

	agentBinGz, ok := AgentHttpBinGz[osArch]
	if !ok {
		return nil, fmt.Errorf("%s not supported by agent", osArch)
	}
	return agentBinGz, nil
}

func NewHttpAgent(ctx context.Context, hst host.Host) (*AgentHttpClient, error) {
	ctx, _ = log.MustContextLoggerSection(ctx, "🐈 Agent")

	agentPath, err := getTmpFile(ctx, hst, "resonance_agent_http")
	if err != nil {
		return nil, err
	}

	if err := hst.Chmod(ctx, agentPath, 0755); err != nil {
		return nil, err
	}

	agentBinGz, err := getHttpAgentBinGz(ctx, hst)
	if err != nil {
		return nil, err
	}

	agentReader, err := gzip.NewReader(bytes.NewReader(agentBinGz))
	if err != nil {
		return nil, err
	}

	if err := copyReader(ctx, hst, agentReader, agentPath); err != nil {
		return nil, err
	}

	agent := AgentHttpClient{
		Host:       hst,
		path:       agentPath,
		spawnErrCh: make(chan error),
	}

	if err := agent.spawn(ctx); err != nil {
		return nil, err
	}

	return &agent, nil
}

func (a *AgentHttpClient) spawn(ctx context.Context) error {
	logger := log.MustLogger(ctx)

	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}

	a.Client = &http.Client{
		Transport: &http2.Transport{
			DialTLSContext: func(
				ctx context.Context, network, addr string, cfg *tls.Config,
			) (net.Conn, error) {
				return aNet.IOConn{
					Reader: stdoutReader,
					Writer: stdinWriter,
				}, nil
			},
			AllowHTTP: true,
		},
	}

	go func() {
		waitStatus, runErr := a.Host.Run(ctx, host.Cmd{
			Path:   a.path,
			Stdin:  stdinReader,
			Stdout: stdoutWriter,
			Stderr: writerLogger{
				Logger: logger,
			},
		})
		var waitStatusErr error
		if !waitStatus.Success() {
			waitStatusErr = errors.New(waitStatus.String())
		}
		stdinWriterErr := stdinWriter.Close()
		stdinReaderErr := stdinReader.Close()
		stdoutWriterErr := stdoutWriter.Close()
		stdoutReaderErr := stdoutReader.Close()
		a.spawnErrCh <- multierr.Combine(
			runErr,
			waitStatusErr,
			stdinWriterErr,
			stdinReaderErr,
			stdoutWriterErr,
			stdoutReaderErr,
		)
	}()

	resp, err := a.get("/ping")
	if err != nil {
		return multierr.Combine(err, a.Close(ctx))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return multierr.Combine(err, a.Close(ctx))
	}
	if string(bodyBytes) != "Pong" {
		return multierr.Combine(
			fmt.Errorf("pinging agent failed: unexpected body %#v", string(bodyBytes)),
			a.Close(ctx),
		)
	}

	return nil
}

func (a AgentHttpClient) checkResponseStatus(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	} else if resp.StatusCode == http.StatusInternalServerError {
		decoder := json.NewDecoder(resp.Body)
		decoder.DisallowUnknownFields()
		var apiErr api.Error
		if err := decoder.Decode(&apiErr); err != nil {
			return err
		}
		return apiErr.Error()
	} else {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unexpected status code %d: failed to read body: %s", resp.StatusCode, err)
		}
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

func (a AgentHttpClient) unmarshalResponse(resp *http.Response, bodyInterface interface{}) error {
	decoder := json.NewDecoder(resp.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(bodyInterface); err != nil {
		return err
	}
	return nil
}

func (a AgentHttpClient) get(path string) (*http.Response, error) {
	resp, err := a.Client.Get(fmt.Sprintf("http://agent%s", path))
	if err != nil {
		return nil, err
	}

	if err := a.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a AgentHttpClient) post(path string, bodyInterface interface{}) (*http.Response, error) {
	url := fmt.Sprintf("http://agent%s", path)

	body := &bytes.Buffer{}
	encoder := json.NewEncoder(body)
	if err := encoder.Encode(bodyInterface); err != nil {
		return nil, err
	}

	resp, err := a.Client.Post(url, "application/json", body)
	if err != nil {
		return nil, err
	}

	return resp, a.checkResponseStatus(resp)
}

func (a AgentHttpClient) delete(path string) (*http.Response, error) {
	url := fmt.Sprintf("http://agent%s", path)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if err := a.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a AgentHttpClient) put(path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("http://agent%s", path)
	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if err := a.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a AgentHttpClient) Geteuid(ctx context.Context) (uint64, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Geteuid")

	resp, err := a.get("/uid")
	if err != nil {
		return 0, err
	}

	var uid uint64
	if err := a.unmarshalResponse(resp, &uid); err != nil {
		return 0, err
	}

	return uid, nil
}

func (a AgentHttpClient) Getegid(ctx context.Context) (uint64, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Getegid")

	resp, err := a.get("/gid")
	if err != nil {
		return 0, err
	}

	var gid uint64
	if err := a.unmarshalResponse(resp, &gid); err != nil {
		return 0, err
	}

	return gid, nil
}

func (a AgentHttpClient) Chmod(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Chmod", "name", name, "mode", mode)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Chmod",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	_, err := a.post(fmt.Sprintf("/file%s", name), api.File{
		Action: api.Chmod,
		Mode:   mode,
	})

	return err
}

func (a AgentHttpClient) Chown(ctx context.Context, name string, uid, gid uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Chown", "name", name, "uid", uid, "gid", gid)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Chown",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	_, err := a.post(fmt.Sprintf("/file%s", name), api.File{
		Action: api.Chown,
		Uid:    uid,
		Gid:    gid,
	})

	return err
}

func (a AgentHttpClient) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lookup", "username", username)

	resp, err := a.get(fmt.Sprintf("/user/%s", username))
	if err != nil {
		return nil, err
	}

	var u user.User
	if err := a.unmarshalResponse(resp, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (a AgentHttpClient) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("LookupGroup", "name", name)

	resp, err := a.get(fmt.Sprintf("/group/%s", name))
	if err != nil {
		return nil, err
	}

	var g user.Group
	if err := a.unmarshalResponse(resp, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

func (a AgentHttpClient) Lstat(ctx context.Context, name string) (*host.Stat_t, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lstat", "name", name)

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "Lstat",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	resp, err := a.get(fmt.Sprintf("/file%s?lstat=true", name))
	if err != nil {
		return nil, err
	}

	var stat_t host.Stat_t
	if err := a.unmarshalResponse(resp, &stat_t); err != nil {
		return nil, err
	}

	return &stat_t, nil
}

func (a AgentHttpClient) ReadDir(ctx context.Context, name string) ([]host.DirEnt, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("ReadDir", "name", name)

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "ReadDir",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	resp, err := a.get(fmt.Sprintf("/file%s?read_dir=true", name))
	if err != nil {
		return nil, err
	}

	dirEnts := []host.DirEnt{}
	if err := a.unmarshalResponse(resp, &dirEnts); err != nil {
		return nil, err
	}

	return dirEnts, nil
}

func (a AgentHttpClient) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("ReadFile", "name", name)

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "ReadFile",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	resp, err := a.get(fmt.Sprintf("/file%s", name))
	if err != nil {
		return nil, err
	}

	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return contents, nil
}

func (a AgentHttpClient) Symlink(ctx context.Context, oldname, newname string) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Symlink", "oldname", oldname, "newname", newname)

	if !filepath.IsAbs(newname) {
		return &fs.PathError{
			Op:   "Symlink",
			Path: newname,
			Err:  errors.New("path must be absolute"),
		}
	}

	_, err := a.post(fmt.Sprintf("/file%s", newname), api.File{
		Action:  api.Symlink,
		Oldname: oldname,
	})

	return err
}

func (a AgentHttpClient) Readlink(ctx context.Context, name string) (string, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Readlink", "name", name)

	if !filepath.IsAbs(name) {
		return "", &fs.PathError{
			Op:   "Readlink",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	resp, err := a.get(fmt.Sprintf("/file%s?readlink=true", name))
	if err != nil {
		return "", err
	}

	var link string
	if err := a.unmarshalResponse(resp, &link); err != nil {
		return "", err
	}

	return link, nil
}

func (a AgentHttpClient) Mkdir(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Mkdir", "name", name)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Mkdir",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	_, err := a.post(fmt.Sprintf("/file%s", name), api.File{
		Action: api.Mkdir,
		Mode:   mode,
	})

	return err
}

func (a AgentHttpClient) Remove(ctx context.Context, name string) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Remove", "name", name)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "Remove",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	_, err := a.delete(fmt.Sprintf("/file%s", name))
	if err != nil {
		return err
	}

	return nil
}

func (a AgentHttpClient) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Run", "cmd", cmd)

	var stdin []byte
	if cmd.Stdin != nil {
		var err error
		stdin, err = io.ReadAll(cmd.Stdin)
		if err != nil {
			return host.WaitStatus{}, err
		}
	}

	var stdout bool
	if cmd.Stdout != nil {
		stdout = true
	}

	var stderr bool
	if cmd.Stderr != nil {
		stderr = true
	}

	resp, err := a.post("/run", api.Cmd{
		Path:   cmd.Path,
		Args:   cmd.Args,
		Env:    cmd.Env,
		Dir:    cmd.Dir,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return host.WaitStatus{}, err
	}

	var cs api.CmdResponse
	if err := a.unmarshalResponse(resp, &cs); err != nil {
		return host.WaitStatus{}, err
	}

	if cmd.Stdout != nil {
		_, err := io.Copy(cmd.Stdout, bytes.NewReader(cs.Stdout))
		if err != nil {
			return host.WaitStatus{}, err
		}
	}

	if cmd.Stderr != nil {
		_, err := io.Copy(cmd.Stderr, bytes.NewReader(cs.Stderr))
		if err != nil {
			return host.WaitStatus{}, err
		}
	}

	return cs.WaitStatus, nil
}

func (a AgentHttpClient) WriteFile(ctx context.Context, name string, data []byte, mode uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("WriteFile", "name", name, "data", data, "mode", mode)

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "WriteFile",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	_, err := a.put(fmt.Sprintf("/file%s?mode=%d", name, mode), bytes.NewReader(data))
	if err != nil {
		return err
	}

	return nil
}

func (a AgentHttpClient) String() string {
	return a.Host.String()
}

func (a AgentHttpClient) Type() string {
	return a.Host.Type()
}

func (a *AgentHttpClient) Close(ctx context.Context) error {
	_, shutdownErr := a.post("/shutdown", nil)

	var spawnErr error
	if shutdownErr == nil {
		spawnErr = <-a.spawnErrCh
	}

	a.Client.CloseIdleConnections()

	hostErr := a.Host.Close(ctx)

	return multierr.Combine(
		shutdownErr,
		spawnErr,
		hostErr,
	)
}
