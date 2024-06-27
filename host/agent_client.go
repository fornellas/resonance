package host

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"go.uber.org/multierr"
	"gopkg.in/yaml.v3"

	"github.com/sirupsen/logrus"

	"github.com/fornellas/resonance/host/agent/api"
	aNet "github.com/fornellas/resonance/host/agent/net"
	"github.com/fornellas/resonance/host/types"

	"github.com/alessio/shellescape"
	"golang.org/x/net/http2"

	"github.com/fornellas/resonance/log"
)

var AgentBinGz = map[string][]byte{}

// AgentClient interacts with a given Host using an agent that's copied and ran at the
// host.
type AgentClient struct {
	Host   Host
	Path   string
	Client *http.Client
	waitCn chan struct{}
}

func (a AgentClient) checkResponseStatus(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	} else if resp.StatusCode == http.StatusInternalServerError {
		decoder := yaml.NewDecoder(resp.Body)
		decoder.KnownFields(true)
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

func (a AgentClient) unmarshalResponse(resp *http.Response, bodyInterface interface{}) error {
	decoder := yaml.NewDecoder(resp.Body)
	decoder.KnownFields(true)
	if err := decoder.Decode(bodyInterface); err != nil {
		return err
	}
	return nil
}

func (a AgentClient) get(path string) (*http.Response, error) {
	resp, err := a.Client.Get(fmt.Sprintf("http://agent%s", path))
	if err != nil {
		return nil, err
	}

	if err := a.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a AgentClient) post(path string, bodyInterface interface{}) (*http.Response, error) {
	url := fmt.Sprintf("http://agent%s", path)

	contentType := "application/yaml"

	bodyData, err := yaml.Marshal(bodyInterface)
	if err != nil {
		return nil, err
	}
	body := bytes.NewBuffer(bodyData)

	resp, err := a.Client.Post(url, contentType, body)
	if err != nil {
		return nil, err
	}

	return resp, a.checkResponseStatus(resp)
}

func (a AgentClient) delete(path string) (*http.Response, error) {
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

func (a AgentClient) put(path string, body io.Reader) (*http.Response, error) {
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

func (a AgentClient) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Chmod %v %s", mode, name)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	_, err := a.post(fmt.Sprintf("/file%s", name), api.File{
		Action: api.Chmod,
		Mode:   mode,
	})

	return err
}

func (a AgentClient) Chown(ctx context.Context, name string, uid, gid int) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Chown %v %v %s", uid, gid, name)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	_, err := a.post(fmt.Sprintf("/file%s", name), api.File{
		Action: api.Chown,
		Uid:    uid,
		Gid:    gid,
	})

	return err
}

func (a AgentClient) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Lookup %s", username)

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

func (a AgentClient) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("LookupGroup %s", name)

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

func (a AgentClient) Lstat(ctx context.Context, name string) (types.HostFileInfo, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Lstat %s", name)

	if !filepath.IsAbs(name) {
		return types.HostFileInfo{}, fmt.Errorf("path must be absolute: %s", name)
	}

	resp, err := a.get(fmt.Sprintf("/file%s?lstat=true", name))
	if err != nil {
		return types.HostFileInfo{}, err
	}

	var hfi types.HostFileInfo
	if err := a.unmarshalResponse(resp, &hfi); err != nil {
		return types.HostFileInfo{}, err
	}
	hfi.ModTime = hfi.ModTime.Local()
	return hfi, nil
}

func (a AgentClient) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Mkdir %s", name)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	_, err := a.post(fmt.Sprintf("/file%s", name), api.File{
		Action: api.Mkdir,
		Mode:   perm,
	})

	return err
}

func (a AgentClient) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("ReadFile %s", name)

	if !filepath.IsAbs(name) {
		return nil, fmt.Errorf("path must be absolute: %s", name)
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

func (a AgentClient) Remove(ctx context.Context, name string) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Remove %s", name)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	_, err := a.delete(fmt.Sprintf("/file%s", name))
	if err != nil {
		return err
	}

	return nil
}

func (a AgentClient) Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Run %s", cmd)

	var stdin []byte
	if cmd.Stdin != nil {
		var err error
		stdin, err = io.ReadAll(cmd.Stdin)
		if err != nil {
			return types.WaitStatus{}, err
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
		return types.WaitStatus{}, err
	}

	var cs api.CmdResponse
	if err := a.unmarshalResponse(resp, &cs); err != nil {
		return types.WaitStatus{}, err
	}

	if cmd.Stdout != nil {
		_, err := io.Copy(cmd.Stdout, bytes.NewReader(cs.Stdout))
		if err != nil {
			return types.WaitStatus{}, err
		}
	}

	if cmd.Stderr != nil {
		_, err := io.Copy(cmd.Stderr, bytes.NewReader(cs.Stderr))
		if err != nil {
			return types.WaitStatus{}, err
		}
	}

	return cs.WaitStatus, nil
}

func (a AgentClient) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("WriteFile %s %v", name, perm)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	_, err := a.put(fmt.Sprintf("/file%s?perm=%d", name, perm), bytes.NewReader(data))
	if err != nil {
		return err
	}

	return nil
}

func (a AgentClient) String() string {
	return a.Host.String()
}

func (a *AgentClient) Close() error {
	a.post("/shutdown", nil)
	a.Client.CloseIdleConnections()
	<-a.waitCn
	return a.Host.Close()
}

type writerLogger struct {
	Logger *logrus.Logger
}

func (wl writerLogger) Write(b []byte) (int, error) {
	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		if len(line) == 0 && i+1 == len(lines) {
			break
		}
		wl.Logger.Errorf("Agent: %s", line)
	}
	return len(b), nil
}

func (a *AgentClient) spawn(ctx context.Context) error {
	logger := log.GetLogger(ctx)

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
				return aNet.Conn{
					Reader: stdoutReader,
					Writer: stdinWriter,
				}, nil
			},
			AllowHTTP: true,
		},
	}

	go func() {
		defer func() { a.waitCn <- struct{}{} }()
		waitStatus, err := a.Host.Run(ctx, types.Cmd{
			Path:   a.Path,
			Stdin:  stdinReader,
			Stdout: stdoutWriter,
			Stderr: writerLogger{
				Logger: logger,
			},
		})
		if err != nil {
			logger.Errorf("failed to run agent: %s", err)
		}
		if !waitStatus.Success() {
			logger.Errorf("agent exited with error: %s", waitStatus.String())
		}
		stdinWriter.Close()
		stdoutReader.Close()
		stdinReader.Close()
		stdoutWriter.Close()
	}()

	resp, err := a.get("/ping")
	if err != nil {
		return multierr.Combine(err, a.Close())
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return multierr.Combine(err, a.Close())
	}
	if string(bodyBytes) != "Pong" {
		return multierr.Combine(
			fmt.Errorf("pinging agent failed: unexpected body %#v", string(bodyBytes)),
			a.Close(),
		)
	}

	return nil
}

func getGoArch(machine string) (string, error) {
	matched, err := regexp.MatchString("^i[23456]86$", machine)
	if err != nil {
		panic(err)
	}
	if matched {
		return "386", nil
	}
	matched, err = regexp.MatchString("^x86_64$", machine)
	if err != nil {
		panic(err)
	}
	if matched {
		return "amd64", nil
	}
	matched, err = regexp.MatchString("^armv6l|armv7l$", machine)
	if err != nil {
		panic(err)
	}
	if matched {
		return "arm", nil
	}
	matched, err = regexp.MatchString("^aarch64$", machine)
	if err != nil {
		panic(err)
	}
	if matched {
		return "arm64", nil
	}
	return "", fmt.Errorf("machine %#v not supported by agent", machine)
}

func getAgentBinGz(ctx context.Context, hst Host) ([]byte, error) {
	cmd := types.Cmd{
		Path: "uname",
		Args: []string{"-m"},
	}
	waitStatus, stdout, stderr, err := Run(ctx, hst, cmd)
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

	agentBinGz, ok := AgentBinGz[osArch]
	if !ok {
		return nil, fmt.Errorf("%s not supported by agent", osArch)
	}
	return agentBinGz, nil
}

func getTmpFile(ctx context.Context, hst Host, template string) (string, error) {
	cmd := types.Cmd{
		Path: "mktemp",
		Args: []string{"-t", fmt.Sprintf("%s.XXXXXXXX", template)},
	}
	waitStatus, stdout, stderr, err := Run(ctx, hst, cmd)
	if err != nil {
		return "", err
	}
	if !waitStatus.Success() {
		return "", fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	return strings.TrimRight(stdout, "\n"), nil
}

func copyReader(ctx context.Context, hst Host, reader io.Reader, path string) error {
	cmd := types.Cmd{
		Path:  "sh",
		Args:  []string{"-c", fmt.Sprintf("cat > %s", shellescape.Quote(path))},
		Stdin: reader,
	}
	waitStatus, stdout, stderr, err := Run(ctx, hst, cmd)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		return fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdout, stderr,
		)
	}
	return nil
}

func NewAgent(ctx context.Context, hst Host) (*AgentClient, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🐈 Agent")
	nestedCtx := log.IndentLogger(ctx)

	agentPath, err := getTmpFile(nestedCtx, hst, "resonance_agent")
	if err != nil {
		return nil, err
	}

	if err := hst.Chmod(nestedCtx, agentPath, os.FileMode(0755)); err != nil {
		return nil, err
	}

	agentBinGz, err := getAgentBinGz(nestedCtx, hst)
	if err != nil {
		return nil, err
	}

	agentReader, err := gzip.NewReader(bytes.NewReader(agentBinGz))
	if err != nil {
		return nil, err
	}

	if err := copyReader(nestedCtx, hst, agentReader, agentPath); err != nil {
		return nil, err
	}

	agent := AgentClient{
		Host:   hst,
		Path:   agentPath,
		waitCn: make(chan struct{}),
	}

	if err := agent.spawn(ctx); err != nil {
		return nil, err
	}

	return &agent, nil
}
