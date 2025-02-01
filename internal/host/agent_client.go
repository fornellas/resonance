package host

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"os/user"
	"regexp"
	"strings"

	"al.essio.dev/pkg/shellescape"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/internal/host/agent_server/proto"
	"github.com/fornellas/resonance/internal/host/lib"
	"github.com/fornellas/resonance/log"
)

var ErrAgentUnsupportedOsArch = fmt.Errorf("OS and architecture not supported")

func getGrpcError(err error) error {
	if status, ok := status.FromError(err); ok {
		switch status.Code() {
		case codes.PermissionDenied:
			return os.ErrPermission
		case codes.NotFound:
			return os.ErrNotExist
		case codes.AlreadyExists:
			return fs.ErrExist
		}
	}
	return err
}

type AgentClientReadFileReadCloser struct {
	Stream     grpc.ServerStreamingClient[proto.ReadFileResponse]
	CancelFunc context.CancelFunc
	Data       []byte
}

func (r *AgentClientReadFileReadCloser) Read(p []byte) (int, error) {
	if len(r.Data) > 0 {
		n := copy(p, r.Data)
		if n < len(r.Data) {
			r.Data = r.Data[n:]
		} else {
			r.Data = nil
		}
		return n, nil
	}

	readFileResponse, err := r.Stream.Recv()
	if err != nil {
		if err == io.EOF {
			return 0, err
		}
		return 0, getGrpcError(err)
	}

	n := copy(p, readFileResponse.Chunk)
	if n < len(readFileResponse.Chunk) {
		r.Data = readFileResponse.Chunk[n:]
	} else {
		r.Data = nil
	}

	return n, nil
}

func (r *AgentClientReadFileReadCloser) Close() error {
	r.CancelFunc()
	return nil
}

var AgentGrpcBinGz = map[string][]byte{}

type WriterLogger struct {
	Logger *slog.Logger
}

func (wl WriterLogger) Write(b []byte) (int, error) {
	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		if len(line) == 0 && i+1 == len(lines) {
			break
		}
		wl.Logger.Error("Agent", "line", line)
	}
	return len(b), nil
}

var AgentBinGz = map[string][]byte{}

// AgentClient interacts with a given Host using an agent that's copied and ran at the
// host.
type AgentClient struct {
	Host              host.Host
	path              string
	grpcClientConn    *grpc.ClientConn
	hostServiceClient proto.HostServiceClient
	spawnErrCh        chan error
}

func getTmpFile(ctx context.Context, hst host.Host, template string) (string, error) {
	cmd := host.Cmd{
		Path: "mktemp",
		Args: []string{"-t", fmt.Sprintf("%s.XXXXXXXX", template)},
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, hst, cmd)
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

func getAgentBinGz(ctx context.Context, hst host.Host) ([]byte, error) {
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

	agentBinGz, ok := AgentBinGz[osArch]
	if !ok {
		return nil, ErrAgentUnsupportedOsArch
	}
	return agentBinGz, nil
}

func copyReader(ctx context.Context, hst host.Host, reader io.Reader, path string) error {
	cmd := host.Cmd{
		Path:  "sh",
		Args:  []string{"-c", fmt.Sprintf("cat > %s", shellescape.Quote(path))},
		Stdin: reader,
	}
	waitStatus, stdout, stderr, err := host.Run(ctx, hst, cmd)
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

func NewAgentClient(ctx context.Context, hst host.Host) (*AgentClient, error) {
	ctx, _ = log.MustContextLoggerSection(ctx, "🐈 Agent")

	agentPath, err := getTmpFile(ctx, hst, "resonance_agent")
	if err != nil {
		return nil, err
	}

	if err := hst.Chmod(ctx, agentPath, 0755); err != nil {
		return nil, err
	}

	agentBinGz, err := getAgentBinGz(ctx, hst)
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

	agent := AgentClient{
		Host:       hst,
		path:       agentPath,
		spawnErrCh: make(chan error),
	}

	if err := agent.spawn(ctx); err != nil {
		return nil, err
	}

	return &agent, nil
}

func (a *AgentClient) spawn(ctx context.Context) error {
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
	a.grpcClientConn, err = grpc.NewClient(
		"127.0.0.1",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return lib.IOConn{
				Reader: stdoutReader,
				Writer: stdinWriter,
			}, nil
		}),
	)
	if err != nil {
		return err
	}

	go func() {
		waitStatus, runErr := a.Host.Run(ctx, host.Cmd{
			Path:   a.path,
			Stdin:  stdinReader,
			Stdout: stdoutWriter,
			Stderr: WriterLogger{
				Logger: logger,
			},
		})
		var waitStatusErr error
		if !waitStatus.Success() {
			waitStatusErr = errors.New(waitStatus.String())
		}
		a.spawnErrCh <- errors.Join(
			runErr,
			waitStatusErr,
		)
	}()

	a.hostServiceClient = proto.NewHostServiceClient(a.grpcClientConn)
	resp, err := a.hostServiceClient.Ping(ctx, &proto.PingRequest{})

	if err != nil {
		return errors.Join(err, a.Close(ctx))
	}

	if resp.Message != "Pong" {
		defer a.Close(ctx)
		return fmt.Errorf("unexpected response from agent: %s", resp.Message)
	}

	return nil
}

func (a *AgentClient) Geteuid(ctx context.Context) (uint64, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Geteuid")

	getuidResponse, err := a.hostServiceClient.Geteuid(ctx, &proto.Empty{})
	if err != nil {
		return 0, err
	}

	return getuidResponse.Uid, nil
}

func (a *AgentClient) Getegid(ctx context.Context) (uint64, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Getegid")

	getgidResponse, err := a.hostServiceClient.Getegid(ctx, &proto.Empty{})
	if err != nil {
		return 0, err
	}

	return getgidResponse.Gid, nil
}

func (a *AgentClient) Chmod(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Chmod", "name", name, "mode", mode)

	_, err := a.hostServiceClient.Chmod(ctx, &proto.ChmodRequest{
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

func (a *AgentClient) Chown(ctx context.Context, name string, uid, gid uint32) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Chown", "name", name, "uid", uid, "gid", gid)

	_, err := a.hostServiceClient.Chown(ctx, &proto.ChownRequest{
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

func (a *AgentClient) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lookup", "username", username)

	resp, err := a.hostServiceClient.Lookup(ctx, &proto.LookupRequest{
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

func (a *AgentClient) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("LookupGroup", "group", name)

	resp, err := a.hostServiceClient.LookupGroup(ctx, &proto.LookupGroupRequest{
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

func (a *AgentClient) Lstat(ctx context.Context, name string) (*host.Stat_t, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lstat", "name", name)

	resp, err := a.hostServiceClient.Lstat(ctx, &proto.LstatRequest{
		Name: name,
	})
	if err != nil {
		return nil, getGrpcError(err)
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

func (a *AgentClient) ReadDir(ctx context.Context, name string) ([]host.DirEnt, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("ReadDir", "name", name)

	resp, err := a.hostServiceClient.ReadDir(ctx, &proto.ReadDirRequest{
		Name: name,
	})
	if err != nil {
		return nil, getGrpcError(err)
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

func (a *AgentClient) Mkdir(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Mkdir", "name", name)

	_, err := a.hostServiceClient.Mkdir(ctx, &proto.MkdirRequest{
		Name: name,
		Mode: mode,
	})
	if err != nil {
		return getGrpcError(err)
	}

	return nil
}

func (a *AgentClient) ReadFile(ctx context.Context, name string) (io.ReadCloser, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("ReadFile", "name", name)

	ctx, cancelFunc := context.WithCancel(ctx)

	stream, err := a.hostServiceClient.ReadFile(ctx, &proto.ReadFileRequest{Name: name})
	if err != nil {
		cancelFunc()
		return nil, getGrpcError(err)
	}

	// ReadFile will succeeds to create the stream before the server function is called.
	// Because of this, we require to read the first element of the stream here, as it
	// enables to catch the various errors we're expected to return.
	readFileResponse, err := stream.Recv()
	if err != nil {
		cancelFunc()
		return nil, getGrpcError(err)
	}

	return &AgentClientReadFileReadCloser{
		Stream:     stream,
		CancelFunc: cancelFunc,
		Data:       readFileResponse.Chunk,
	}, nil
}

func (a *AgentClient) Symlink(ctx context.Context, oldname, newname string) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Symlink", "oldname", oldname, "newname", newname)

	_, err := a.hostServiceClient.Symlink(ctx, &proto.SymlinkRequest{
		Oldname: oldname,
		Newname: newname,
	})

	if err != nil {
		return getGrpcError(err)
	}

	return nil
}

func (a *AgentClient) Readlink(ctx context.Context, name string) (string, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Readlink", "name", name)

	resp, err := a.hostServiceClient.ReadLink(ctx, &proto.ReadLinkRequest{
		Name: name,
	})

	if err != nil {
		return "", getGrpcError(err)
	}

	return resp.Destination, nil
}

func (a *AgentClient) Remove(ctx context.Context, name string) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Remove", "name", name)

	_, err := a.hostServiceClient.Remove(ctx, &proto.RemoveRequest{
		Name: name,
	})
	if err != nil {
		return getGrpcError(err)
	}

	return nil
}

func (a *AgentClient) Mknod(ctx context.Context, pathName string, mode uint32, dev uint64) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Mknod", "pathName", pathName, "mode", mode, "dev", dev)

	_, err := a.hostServiceClient.Mknod(ctx, &proto.MknodRequest{
		PathName: pathName,
		Mode:     mode,
		Dev:      dev,
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

func (a *AgentClient) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
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

	resp, err := a.hostServiceClient.Run(ctx, &proto.RunRequest{
		Path:    cmd.Path,
		Args:    cmd.Args,
		EnvVars: cmd.Env,
		Dir:     cmd.Dir,
		Stdin:   stdin,
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

func (a *AgentClient) WriteFile(ctx context.Context, name string, data []byte, perm uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("WriteFile", "name", name, "data", data, "perm", perm)

	_, err := a.hostServiceClient.WriteFile(ctx, &proto.WriteFileRequest{
		Name: name,
		Data: data,
		Perm: perm,
	})

	if err != nil {
		return getGrpcError(err)
	}

	return nil
}

func (a *AgentClient) String() string {
	return a.Host.String()
}

func (a *AgentClient) Type() string {
	return a.Host.Type()
}

func (a *AgentClient) Close(ctx context.Context) error {

	_, shutdownErr := a.hostServiceClient.Shutdown(ctx, &proto.Empty{})

	var spawnErr error
	if shutdownErr == nil {
		spawnErr = <-a.spawnErrCh
	}

	grpcClientConnErr := a.grpcClientConn.Close()

	hostCloseErr := a.Host.Close(ctx)

	return errors.Join(
		shutdownErr,
		grpcClientConnErr,
		spawnErr,
		hostCloseErr,
	)
}
