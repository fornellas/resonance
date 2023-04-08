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

	"gopkg.in/yaml.v3"

	"github.com/sirupsen/logrus"

	"github.com/fornellas/resonance/host/agent/api"
	aNet "github.com/fornellas/resonance/host/agent/net"

	"github.com/alessio/shellescape"
	"golang.org/x/net/http2"

	"github.com/fornellas/resonance/log"
)

var AgentBinGz = map[string][]byte{}

// Agent interacts with a given Host using an agent that's copied and ran at the
// host.
type Agent struct {
	Host   Host
	Path   string
	Client *http.Client
}

func (a Agent) checkResponseStatus(resp *http.Response) error {
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

func (a Agent) unmarshalResponse(resp *http.Response, bodyInterface interface{}) error {
	decoder := yaml.NewDecoder(resp.Body)
	decoder.KnownFields(true)
	if err := decoder.Decode(bodyInterface); err != nil {
		return err
	}
	return nil
}

func (a Agent) get(path string) (*http.Response, error) {
	resp, err := a.Client.Get(fmt.Sprintf("http://agent%s", path))
	if err != nil {
		return nil, err
	}

	if err := a.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a Agent) post(path string, bodyInterface interface{}) error {
	url := fmt.Sprintf("http://agent%s", path)

	contentType := "application/yaml"

	bodyData, err := yaml.Marshal(bodyInterface)
	if err != nil {
		return err
	}
	body := bytes.NewBuffer(bodyData)

	resp, err := a.Client.Post(url, contentType, body)
	if err != nil {
		return err
	}

	return a.checkResponseStatus(resp)
}

func (a Agent) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Chmod %v %s", mode, name)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	return a.post(fmt.Sprintf("/file%s", name), api.File{
		Action: api.Chmod,
		Mode:   mode,
	})
}

func (a Agent) Chown(ctx context.Context, name string, uid, gid int) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Chown %v %v %s", uid, gid, name)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	return a.post(fmt.Sprintf("/file%s", name), api.File{
		Action: api.Chown,
		Uid:    uid,
		Gid:    gid,
	})
}

func (a Agent) Lookup(ctx context.Context, username string) (*user.User, error) {
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

func (a Agent) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
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

func (a Agent) Lstat(ctx context.Context, name string) (HostFileInfo, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Lstat %s", name)
	return HostFileInfo{}, fmt.Errorf("TODO Agent.Lstat")
}

func (a Agent) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Mkdir %s", name)
	return fmt.Errorf("TODO Agent.Mkdir")
}

func (a Agent) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("ReadFile %s", name)
	return nil, fmt.Errorf("TODO Agent.ReadFile")
	// logger := log.GetLogger(ctx)
	// logger.Debugf("ReadFile %s", name)

	// resp, err := a.get(fmt.Sprintf("/file/%s", name))
	// // resp, err := a.get(fmt.Sprintf("/file/%s", url.QueryEscape(name)))
	// if err != nil {
	// 	return nil, err
	// }
	// if resp.StatusCode != http.StatusOK {
	// 	return nil, fmt.Errorf("status code %d", resp.StatusCode)
	// }
	// return io.ReadAll(resp.Body)
}

func (a Agent) Remove(ctx context.Context, name string) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Remove %s", name)
	return fmt.Errorf("TODO Agent.Remove")
}

func (a Agent) Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Run %s", cmd)
	return WaitStatus{}, fmt.Errorf("TODO Agent.Run")
}

func (a Agent) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("WriteFile %s %v", name, perm)
	return fmt.Errorf("TODO Agent.WriteFile")
}

func (a Agent) String() string {
	return a.Host.String()
}

func (a Agent) Close() error {
	// TODO tell agent to exit
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

func (a *Agent) spawnAgent(ctx context.Context) error {
	logger := log.GetLogger(ctx)

	// TODO handle closing
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}

	// TODO handle closing
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
		waitStatus, err := a.Host.Run(ctx, Cmd{
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
	}()

	resp, err := a.get("/ping")
	if err != nil {
		// TODO handle stop agent
		return err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		// TODO handle stop agent
		return err
	}
	if string(bodyBytes) != "Pong" {
		// TODO handle stop agent
		return fmt.Errorf("pinging agent failed: unexpected body %#v", string(bodyBytes))
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
	cmd := Cmd{
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
	cmd := Cmd{
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
	cmd := Cmd{
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

func NewAgent(ctx context.Context, hst Host) (*Agent, error) {
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

	agent := Agent{
		Host: hst,
		Path: agentPath,
	}

	if err := agent.spawnAgent(ctx); err != nil {
		return nil, err
	}

	return &agent, nil
}
