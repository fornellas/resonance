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
	"regexp"

	"github.com/sirupsen/logrus"

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
	Client http.Client
}

func (a Agent) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	return fmt.Errorf("TODO Agent.Chmod")
}

func (a Agent) Chown(ctx context.Context, name string, uid, gid int) error {
	return fmt.Errorf("TODO Agent.Chown")
}

func (a Agent) Lookup(ctx context.Context, username string) (*user.User, error) {
	return nil, fmt.Errorf("TODO Agent.Lookup")
}

func (a Agent) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	return nil, fmt.Errorf("TODO Agent.LookupGroup")
}

func (a Agent) Lstat(ctx context.Context, name string) (HostFileInfo, error) {
	return HostFileInfo{}, fmt.Errorf("TODO Agent.Lstat")
}

func (a Agent) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return fmt.Errorf("TODO Agent.Mkdir")
}

func (a Agent) ReadFile(ctx context.Context, name string) ([]byte, error) {
	return nil, fmt.Errorf("TODO Agent.ReadFile")
}

func (a Agent) Remove(ctx context.Context, name string) error {
	return fmt.Errorf("TODO Agent.Remove")
}

func (a Agent) Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {
	return WaitStatus{}, fmt.Errorf("TODO Agent.Run")
}

func (a Agent) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
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
	wl.Logger.Errorf("Agent: %s", b)
	return len(b), nil
}

func (a Agent) spawnAgent(ctx context.Context) error {
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

	a.Client = http.Client{
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
	}()

	resp, err := a.Client.Get("http://resonance_agent/ping")
	if err != nil {
		// TODO handle stop agent
		return err
	}
	if resp.StatusCode != http.StatusOK {
		// TODO handle stop agent
		return fmt.Errorf("pinging agent failed: status code %d", resp.StatusCode)
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
	return "", fmt.Errorf("machine %s not supported by agent", machine)
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
	goarch, err := getGoArch(stdout)
	if err != nil {
		return nil, err
	}
	osArch := fmt.Sprintf("linux.%s", goarch)

	agentBinGz, ok := AgentBinGz[osArch]
	if !ok {
		return nil, fmt.Errorf("GOOS.GOARCH not supported by agent: %s", osArch)
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
	return stdout, nil
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
