package host

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/user"
	"strings"

	"go.uber.org/multierr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

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

	if err := hst.Chmod(ctx, agentPath, 0755); err != nil {
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
	resp, err := Client.Ping(ctx, &proto.PingRequest{})

	if err != nil {
		return multierr.Combine(err, a.Close())
	}

	if resp.Message != "Pong" {
		defer a.Close()
		return fmt.Errorf("unexpected response from agent: %s", resp.Message)
	}

	return nil
}

func (a AgentGrpcClient) Chmod(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Chmod", "name", name, "mode", mode)

	Client := proto.NewHostServiceClient(a.Client)
	_, err := Client.Chmod(ctx, &proto.ChmodRequest{
		Name: name,
		Mode: mode,
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

func (a AgentGrpcClient) Chown(ctx context.Context, name string, uid, gid uint32) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Chown", "name", name, "uid", uid, "gid", gid)

	Client := proto.NewHostServiceClient(a.Client)
	_, err := Client.Chown(ctx, &proto.ChownRequest{
		Name: name,
		Uid:  int64(uid),
		Gid:  int64(gid),
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
	logger := log.MustLogger(ctx)

	logger.Debug("Lookup", "username", username)

	Client := proto.NewHostServiceClient(a.Client)
	resp, err := Client.Lookup(ctx, &proto.LookupRequest{
		Username: username,
	})

	if err != nil {
		if strings.Contains(err.Error(), "user: unknown user") {
			return nil, user.UnknownUserError(username)
		}
		return nil, err
	}

	return &user.User{
		Uid:      resp.Uid,
		Gid:      resp.Gid,
		Username: resp.Username,
		Name:     resp.Name,
		HomeDir:  resp.Homedir,
	}, nil
}

func (a AgentGrpcClient) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("LookupGroup", "group", name)

	Client := proto.NewHostServiceClient(a.Client)
	resp, err := Client.LookupGroup(ctx, &proto.LookupGroupRequest{
		Name: name,
	})

	if err != nil {
		if strings.Contains(err.Error(), "group: unknown group") {
			return nil, user.UnknownGroupError(name)
		}
		return nil, err
	}

	return &user.Group{
		Gid:  resp.Gid,
		Name: resp.Name,
	}, nil
}

func (a AgentGrpcClient) Lstat(ctx context.Context, name string) (*host.Stat_t, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lstat", "name", name)

	client := proto.NewHostServiceClient(a.Client)
	resp, err := client.Lstat(ctx, &proto.LstatRequest{
		Name: name,
	})
	if err != nil {
		if status, ok := status.FromError(err); ok {
			switch status.Code() {
			case codes.PermissionDenied:
				return nil, fs.ErrPermission
			case codes.NotFound:
				return nil, fs.ErrNotExist
			}
		}
		return nil, err
	}

	stat_t := host.Stat_t{
		Dev:     resp.Dev,
		Ino:     resp.Ino,
		Mode:    resp.Mode,
		Nlink:   uint64(resp.Nlink),
		Uid:     resp.Uid,
		Gid:     resp.Gid,
		Rdev:    resp.Rdev,
		Size:    resp.Size,
		Blksize: int64(resp.Blksize),
		Blocks:  resp.Blocks,
		Atim: host.Timespec{
			Sec:  resp.Atim.Sec,
			Nsec: resp.Atim.Nsec,
		},
		Mtim: host.Timespec{
			Sec:  resp.Mtim.Sec,
			Nsec: resp.Mtim.Nsec,
		},
		Ctim: host.Timespec{
			Sec:  resp.Ctim.Sec,
			Nsec: resp.Ctim.Nsec,
		},
	}

	return &stat_t, nil
}

func (a AgentGrpcClient) ReadDir(ctx context.Context, name string) ([]host.DirEnt, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("ReadDir", "name", name)

	client := proto.NewHostServiceClient(a.Client)
	resp, err := client.ReadDir(ctx, &proto.ReadDirRequest{
		Name: name,
	})

	if err != nil {
		if status, ok := status.FromError(err); ok {
			switch status.Code() {
			case codes.PermissionDenied:
				return nil, fs.ErrPermission
			case codes.NotFound:
				return nil, fs.ErrNotExist
			}
		}
		return nil, err
	}

	dirEnts := []host.DirEnt{}
	for _, protoDirEnt := range resp.Entries {
		dirEnts = append(dirEnts, host.DirEnt{
			Name: protoDirEnt.Name,
			Type: uint8(protoDirEnt.Type),
			Ino:  protoDirEnt.Ino,
		})
	}

	return dirEnts, nil
}

func (a AgentGrpcClient) Mkdir(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Mkdir", "name", name)

	client := proto.NewHostServiceClient(a.Client)
	_, err := client.Mkdir(ctx, &proto.MkdirRequest{
		Name: name,
		Mode: mode,
	})

	if err != nil {
		if status, ok := status.FromError(err); ok {
			switch status.Code() {
			case codes.PermissionDenied:
				return fs.ErrPermission
			case codes.NotFound:
				return fs.ErrNotExist
			case codes.AlreadyExists:
				return fs.ErrExist
			}
		}
		return err
	}

	return nil
}

func (a AgentGrpcClient) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("ReadFile", "name", name)

	client := proto.NewHostServiceClient(a.Client)
	stream, err := client.ReadFile(ctx, &proto.ReadFileRequest{
		Name: name,
	})

	if err != nil {
		if status, ok := status.FromError(err); ok {
			switch status.Code() {
			case codes.PermissionDenied:
				return nil, os.ErrPermission
			case codes.NotFound:
				return nil, os.ErrNotExist
			}
		}
		return nil, err
	}

	var fileData []byte

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			if status, ok := status.FromError(err); ok {
				switch status.Code() {
				case codes.PermissionDenied:
					return nil, os.ErrPermission
				case codes.NotFound:
					return nil, os.ErrNotExist
				}
			}
			return nil, err

		}

		fileData = append(fileData, resp.Chunk...)
	}

	return fileData, nil
}

func (a AgentGrpcClient) Readlink(ctx context.Context, name string) (string, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Readlink", "name", name)

	client := proto.NewHostServiceClient(a.Client)
	resp, err := client.ReadLink(ctx, &proto.ReadLinkRequest{
		Name: name,
	})

	if err != nil {
		if status, ok := status.FromError(err); ok {
			switch status.Code() {
			case codes.PermissionDenied:
				return "", os.ErrPermission
			case codes.NotFound:
				return "", os.ErrNotExist
			}
		}
		return "", err
	}

	return resp.Destination, nil
}

func (a AgentGrpcClient) Remove(ctx context.Context, name string) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Remove", "name", name)

	client := proto.NewHostServiceClient(a.Client)
	_, err := client.Remove(ctx, &proto.RemoveRequest{
		Name: name,
	})

	if err != nil {
		if status, ok := status.FromError(err); ok {
			switch status.Code() {
			case codes.PermissionDenied:
				return fs.ErrPermission
			case codes.NotFound:
				return fs.ErrNotExist
			}
		}
		return err
	}

	return nil
}

func (a AgentGrpcClient) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
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

	client := proto.NewHostServiceClient(a.Client)
	resp, err := client.Run(ctx, &proto.RunRequest{
		Path:  cmd.Path,
		Args:  cmd.Args,
		Env:   cmd.Env,
		Dir:   cmd.Dir,
		Stdin: stdin,
	})

	if err != nil {
		return host.WaitStatus{}, err
	}

	if cmd.Stdout != nil {
		_, err := io.Copy(cmd.Stdout, bytes.NewReader(resp.Stdout))
		if err != nil {
			return host.WaitStatus{}, err
		}
	}

	if cmd.Stderr != nil {
		_, err := io.Copy(cmd.Stderr, bytes.NewReader(resp.Stderr))
		if err != nil {
			return host.WaitStatus{}, err
		}
	}

	return host.WaitStatus{
		ExitCode: int(resp.Waitstatus.Exitcode),
		Exited:   resp.Waitstatus.Exited,
		Signal:   resp.Waitstatus.Signal,
	}, nil
}

func (a AgentGrpcClient) WriteFile(ctx context.Context, name string, data []byte, perm uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("WriteFile", "name", name, "data", data, "perm", perm)

	client := proto.NewHostServiceClient(a.Client)
	_, err := client.WriteFile(ctx, &proto.WriteFileRequest{
		Name: name,
		Data: data,
		Perm: perm,
	})

	if err != nil {
		if status, ok := status.FromError(err); ok {
			switch status.Code() {
			case codes.PermissionDenied:
				return fs.ErrPermission
			case codes.NotFound:
				return fs.ErrNotExist
			}
		}
		return err
	}

	return nil
}

func (a AgentGrpcClient) String() string {
	return a.Host.String()
}

func (a AgentGrpcClient) Type() string {
	return a.Host.Type()
}

func (a *AgentGrpcClient) Close() error {
	fmt.Println("Closing agent")
	return nil
	// to be implemented
}
