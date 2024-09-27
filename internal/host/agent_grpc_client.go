package host

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"go.uber.org/multierr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/internal/host/agent_server_grpc/proto"
	aNet "github.com/fornellas/resonance/internal/host/agent_server_http/net"
	"github.com/fornellas/resonance/log"
)

var AgentGrpcBinGz = map[string][]byte{}

// AgentGrpcClient interacts with a given Host using an agent that's copied and ran at the
// host.
type AgentGrpcClient struct {
	Host   host.Host
	path   string
	Client *grpc.ClientConn
	waitCn chan struct{}
}

func getGrpcAgentBinGz(ctx context.Context, hst host.Host) ([]byte, error) {
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

	agentBinGz, ok := AgentGrpcBinGz[osArch]
	if !ok {
		return nil, fmt.Errorf("%s not supported by agent", osArch)
	}
	return agentBinGz, nil
}

func NewGrpcAgent(ctx context.Context, hst host.Host) (*AgentGrpcClient, error) {
	ctx, _ = log.MustContextLoggerSection(ctx, "🐈 Agent")

	agentPath, err := getTmpFile(ctx, hst, "resonance_agent")
	if err != nil {
		return nil, err
	}

	if err := hst.Chmod(ctx, agentPath, os.FileMode(0755)); err != nil {
		return nil, err
	}

	agentBinGz, err := getGrpcAgentBinGz(ctx, hst)
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

	agent := AgentGrpcClient{
		Host:   hst,
		path:   agentPath,
		waitCn: make(chan struct{}),
	}

	if err := agent.spawn(ctx); err != nil {
		return nil, err
	}

	return &agent, nil
}

func GetDialer(stdoutReader, stdinWriter *os.File) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (net.Conn, error) {
		return aNet.Conn{
			Reader: stdoutReader,
			Writer: stdinWriter,
		}, nil
	}
}

func (a *AgentGrpcClient) spawn(ctx context.Context) error {
	logger := log.MustLogger(ctx)

	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}

	// We just pass "127.0.0.1" to avoid issues with dns resolution, this value is not used
	a.Client, err = grpc.NewClient("127.0.0.1",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(GetDialer(stdoutReader, stdinWriter)))
	if err != nil {
		return err
	}

	go func() {
		defer func() { a.waitCn <- struct{}{} }()
		waitStatus, err := a.Host.Run(ctx, host.Cmd{
			Path:   a.path,
			Stdin:  stdinReader,
			Stdout: stdoutWriter,
			Stderr: writerLogger{
				Logger: logger,
			},
		})
		if err != nil {
			logger.Error("failed to run agent", "err", err)
		}
		if !waitStatus.Success() {
			logger.Error("agent exited with error", "error", waitStatus)
		}
		stdinWriter.Close()
		stdoutReader.Close()
		stdinReader.Close()
		stdoutWriter.Close()
	}()

	Client := proto.NewHostServiceClient(a.Client)
	pingResp, err := Client.Ping(ctx, &proto.PingRequest{})

	if err != nil {
		return multierr.Combine(err, a.Close())
	}

	if pingResp.Message != "Pong" {
		defer a.Close()
		return fmt.Errorf("unexpected response from agent: %s", pingResp.Message)
	}

	return nil
}

func (a AgentGrpcClient) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Chmod", "name", name, "mode", mode)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	Client := proto.NewHostServiceClient(a.Client)
	_, err := Client.Chmod(ctx, &proto.ChmodRequest{
		Name: name,
		Mode: int32(mode),
	})

	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			return fmt.Errorf("permission denied: %w", fs.ErrPermission)
		}
		if strings.Contains(err.Error(), "no such file or directory") {
			return fmt.Errorf("no such file or directory: %w", fs.ErrNotExist)
		}
		return err
	}

	return nil
}

func (a AgentGrpcClient) Chown(ctx context.Context, name string, uid, gid int) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Chown", "name", name, "uid", uid, "gid", gid)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	Client := proto.NewHostServiceClient(a.Client)
	_, err := Client.Chown(ctx, &proto.ChownRequest{
		Name: name,
		Uid:  int32(uid),
		Gid:  int32(gid),
	})

	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			return fmt.Errorf("permission denied: %w", fs.ErrPermission)
		}
		if strings.Contains(err.Error(), "no such file or directory") {
			return fmt.Errorf("no such file or directory: %w", fs.ErrNotExist)
		}
		return err
	}

	return nil
}

func (a AgentGrpcClient) Lookup(ctx context.Context, username string) (*user.User, error) {
	panic("todo lookup")
	// 	logger := log.MustLogger(ctx)

	// 	logger.Debug("Lookup", "username", username)

	// 	resp, err := a.get(fmt.Sprintf("/user/%s", username))
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// var u user.User
	//
	//	if err := a.unmarshalResponse(resp, &u); err != nil {
	//		return nil, err
	//	}
	//
	// return &u, nil
}

func (a AgentGrpcClient) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	panic("todo lookup group")
	// 	logger := log.MustLogger(ctx)

	// 	logger.Debug("LookupGroup", "name", name)

	// 	resp, err := a.get(fmt.Sprintf("/group/%s", name))
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// var g user.Group
	//
	//	if err := a.unmarshalResponse(resp, &g); err != nil {
	//		return nil, err
	//	}
	//
	// return &g, nil
}

func (a AgentGrpcClient) Lstat(ctx context.Context, name string) (host.HostFileInfo, error) {
	panic("todo lstat")
	// 	logger := log.MustLogger(ctx)

	// 	logger.Debug("Lstat", "name", name)

	// 	if !filepath.IsAbs(name) {
	// 		return host.HostFileInfo{}, fmt.Errorf("path must be absolute: %s", name)
	// 	}

	// 	resp, err := a.get(fmt.Sprintf("/file%s?lstat=true", name))
	// 	if err != nil {
	// 		return host.HostFileInfo{}, err
	// }

	// var hfi host.HostFileInfo
	//
	//	if err := a.unmarshalResponse(resp, &hfi); err != nil {
	//		return host.HostFileInfo{}, err
	//	}
	//
	// hfi.ModTime = hfi.ModTime.Local()
	// return hfi, nil
}

func (a AgentGrpcClient) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	panic("todo mkdir")
	// 	logger := log.MustLogger(ctx)

	// 	logger.Debug("Mkdir", "name", name)

	// 	if !filepath.IsAbs(name) {
	// 		return fmt.Errorf("path must be absolute: %s", name)
	// 	}

	// 	_, err := a.post(fmt.Sprintf("/file%s", name), api.File{
	// 		Action: api.Mkdir,
	// 		Mode:   perm,
	// 	})

	// return err
}

func (a AgentGrpcClient) ReadFile(ctx context.Context, name string) ([]byte, error) {
	panic("todo read file")
	// 	logger := log.MustLogger(ctx)

	// 	logger.Debug("ReadFile", "name", name)

	// 	if !filepath.IsAbs(name) {
	// 		return nil, fmt.Errorf("path must be absolute: %s", name)
	// 	}

	// 	resp, err := a.get(fmt.Sprintf("/file%s", name))
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	contents, err := io.ReadAll(resp.Body)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// return contents, nil
}

func (a AgentGrpcClient) Remove(ctx context.Context, name string) error {
	panic("todo remove")
	// 	logger := log.MustLogger(ctx)

	// 	logger.Debug("Remove", "name", name)

	// 	if !filepath.IsAbs(name) {
	// 		return fmt.Errorf("path must be absolute: %s", name)
	// 	}

	// 	_, err := a.delete(fmt.Sprintf("/file%s", name))
	// 	if err != nil {
	// 		return err
	// 	}

	// return nil
}

func (a AgentGrpcClient) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	panic("todo run")
	// 	logger := log.MustLogger(ctx)

	// 	logger.Debug("Run", "cmd", cmd)

	// 	var stdin []byte
	// 	if cmd.Stdin != nil {
	// 		var err error
	// 		stdin, err = io.ReadAll(cmd.Stdin)
	// 		if err != nil {
	// 			return host.WaitStatus{}, err
	// 		}
	// }

	// 	var stdout bool
	// 	if cmd.Stdout != nil {
	// 		stdout = true
	// 	}

	// 	var stderr bool
	// 	if cmd.Stderr != nil {
	// 		stderr = true
	// 	}

	// 	resp, err := a.post("/run", api.Cmd{
	// 		Path:   cmd.Path,
	// 		Args:   cmd.Args,
	// 		Env:    cmd.Env,
	// 		Dir:    cmd.Dir,
	// 		Stdin:  stdin,
	// 		Stdout: stdout,
	// 		Stderr: stderr,
	// 	})
	// 	if err != nil {
	// 		return host.WaitStatus{}, err
	// 	}

	// 	var cs api.CmdResponse
	// 	if err := a.unmarshalResponse(resp, &cs); err != nil {
	// 		return host.WaitStatus{}, err
	// 	}

	// 	if cmd.Stdout != nil {
	// 		_, err := io.Copy(cmd.Stdout, bytes.NewReader(cs.Stdout))
	// 		if err != nil {
	// 			return host.WaitStatus{}, err
	// 		}
	// 	}

	// 	if cmd.Stderr != nil {
	// 		_, err := io.Copy(cmd.Stderr, bytes.NewReader(cs.Stderr))
	// 		if err != nil {
	// 			return host.WaitStatus{}, err
	// 		}
	// 	}

	// return cs.WaitStatus, nil
}

func (a AgentGrpcClient) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	panic("todo write file")
	// 	logger := log.MustLogger(ctx)

	// 	logger.Debug("WriteFile", "name", name, "data", data, "perm", perm)

	// 	if !filepath.IsAbs(name) {
	// 		return fmt.Errorf("path must be absolute: %s", name)
	// 	}

	// 	_, err := a.put(fmt.Sprintf("/file%s?perm=%d", name, perm), bytes.NewReader(data))
	// 	if err != nil {
	// 		return err
	// 	}

	// return nil
}

func (a AgentGrpcClient) String() string {
	return a.Host.String()
}

func (a AgentGrpcClient) Type() string {
	return a.Host.Type()
}

func (a *AgentGrpcClient) Close() error {
	panic("todo close")
	// a.post("/shutdown", nil) // to be implemented
	// a.Client.Close()
	// <-a.waitCn
	// return a.Host.Close()
}
