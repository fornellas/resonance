package host

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	userPkg "os/user"
	"regexp"
	"strings"
	"sync"
	"syscall"

	"al.essio.dev/pkg/shellescape"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/fornellas/resonance/host/lib"
	hostNet "github.com/fornellas/resonance/host/net"

	"github.com/fornellas/resonance/host/agent_server/proto"
	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

var ErrAgentUnsupportedOsArch = fmt.Errorf("OS and architecture not supported")

func unwrapGrpcStatusErrno(err error) error {
	st := status.Convert(err)
	for _, detail := range st.Details() {
		if errno, ok := detail.(*proto.Errno); ok {
			return syscall.Errno(errno.Errno)
		}
	}
	return err
}

type AgentClientWrapperReadFileReadCloser struct {
	Stream     grpc.ServerStreamingClient[proto.ReadFileResponse]
	CancelFunc context.CancelFunc
	Data       []byte
}

func (rc *AgentClientWrapperReadFileReadCloser) Read(p []byte) (int, error) {
	if len(rc.Data) > 0 {
		n := copy(p, rc.Data)
		if n < len(rc.Data) {
			rc.Data = rc.Data[n:]
		} else {
			rc.Data = nil
		}
		return n, nil
	}

	readFileResponse, err := rc.Stream.Recv()
	if err != nil {
		if err == io.EOF {
			return 0, err
		}
		return 0, unwrapGrpcStatusErrno(err)
	}

	n := copy(p, readFileResponse.Chunk)
	if n < len(readFileResponse.Chunk) {
		rc.Data = readFileResponse.Chunk[n:]
	} else {
		rc.Data = nil
	}

	return n, nil
}

func (rc *AgentClientWrapperReadFileReadCloser) Close() error {
	rc.CancelFunc()
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

// AgentClientWrapper wraps a BaseHost and provides a full Host implementation with the
// use of an ephemeral agent.
type AgentClientWrapper struct {
	BaseHost          types.BaseHost
	path              string
	grpcClientConn    *grpc.ClientConn
	hostServiceClient proto.HostServiceClient
	spawnErrCh        chan error
}

func getTmpFile(ctx context.Context, hst types.BaseHost, template string) (string, error) {
	cmd := types.Cmd{
		Path: "mktemp",
		Args: []string{"-t", fmt.Sprintf("%s.XXXXXXXX", template)},
	}
	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, hst, cmd)
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

func chmod(ctx context.Context, baseHost types.BaseHost, name string, mode uint32) error {
	cmd := types.Cmd{
		Path: "chmod",
		Args: []string{fmt.Sprintf("%o", mode), name},
	}
	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, baseHost, cmd)
	if err != nil {
		return err
	}
	if waitStatus.Success() {
		return nil
	}

	return fmt.Errorf(
		"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
		cmd, waitStatus.String(), stdout, stderr,
	)
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

func getAgentBinGz(ctx context.Context, hst types.BaseHost) ([]byte, error) {
	cmd := types.Cmd{
		Path: "uname",
		Args: []string{"-m"},
	}
	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, hst, cmd)
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

func copyReader(ctx context.Context, hst types.BaseHost, reader io.Reader, path string) error {
	cmd := types.Cmd{
		Path:  "sh",
		Args:  []string{"-c", fmt.Sprintf("cat > %s", shellescape.Quote(path))},
		Stdin: reader,
	}
	waitStatus, stdout, stderr, err := lib.SimpleRun(ctx, hst, cmd)
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

func NewAgentClientWrapper(ctx context.Context, baseHost types.BaseHost) (*AgentClientWrapper, error) {
	ctx, _ = log.MustContextLoggerSection(ctx, "🐈 Agent")

	agentPath, err := getTmpFile(ctx, baseHost, "resonance_agent")
	if err != nil {
		return nil, err
	}

	if err := chmod(ctx, baseHost, agentPath, 0755); err != nil {
		return nil, err
	}

	agentBinGz, err := getAgentBinGz(ctx, baseHost)
	if err != nil {
		return nil, err
	}

	agentReader, err := gzip.NewReader(bytes.NewReader(agentBinGz))
	if err != nil {
		return nil, err
	}

	if err := copyReader(ctx, baseHost, agentReader, agentPath); err != nil {
		return nil, err
	}

	agent := AgentClientWrapper{
		BaseHost:   baseHost,
		path:       agentPath,
		spawnErrCh: make(chan error),
	}

	if err := agent.spawn(ctx); err != nil {
		return nil, err
	}

	return &agent, nil
}

func (h *AgentClientWrapper) spawn(ctx context.Context) error {
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
	h.grpcClientConn, err = grpc.NewClient(
		"127.0.0.1",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return hostNet.IOConn{
				Reader: stdoutReader,
				Writer: stdinWriter,
			}, nil
		}),
	)
	if err != nil {
		return err
	}

	go func() {
		waitStatus, runErr := h.BaseHost.Run(ctx, types.Cmd{
			Path:   h.path,
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

		stdinReaderErr := stdinReader.Close()

		stdoutWriterErr := stdoutWriter.Close()

		h.spawnErrCh <- errors.Join(
			runErr,
			waitStatusErr,
			stdinReaderErr,
			stdoutWriterErr,
		)
	}()

	h.hostServiceClient = proto.NewHostServiceClient(h.grpcClientConn)
	resp, err := h.hostServiceClient.Ping(ctx, &proto.PingRequest{})

	if err != nil {
		return errors.Join(err, h.Close(ctx))
	}

	if resp.Message != "Pong" {
		defer h.Close(ctx)
		return fmt.Errorf("unexpected response from agent: %s", resp.Message)
	}

	return nil
}

func (h *AgentClientWrapper) Geteuid(ctx context.Context) (uint64, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Geteuid")

	getuidResponse, err := h.hostServiceClient.Geteuid(ctx, &proto.Empty{})
	if err != nil {
		return 0, unwrapGrpcStatusErrno(err)
	}

	return getuidResponse.Uid, nil
}

func (h *AgentClientWrapper) Getegid(ctx context.Context) (uint64, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Getegid")

	getgidResponse, err := h.hostServiceClient.Getegid(ctx, &proto.Empty{})
	if err != nil {
		return 0, unwrapGrpcStatusErrno(err)
	}

	return getgidResponse.Gid, nil
}

func (h *AgentClientWrapper) Chmod(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Chmod", "name", name, "mode", mode)

	_, err := h.hostServiceClient.Chmod(ctx, &proto.ChmodRequest{
		Name: name,
		Mode: mode,
	})

	if err != nil {
		return unwrapGrpcStatusErrno(err)
	}

	return nil
}

func (h *AgentClientWrapper) Lchown(ctx context.Context, name string, uid, gid uint32) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Lchown", "name", name, "uid", uid, "gid", gid)

	_, err := h.hostServiceClient.Lchown(ctx, &proto.LchownRequest{
		Name: name,
		Uid:  int64(uid),
		Gid:  int64(gid),
	})

	if err != nil {
		return unwrapGrpcStatusErrno(err)
	}

	return nil
}

func (h *AgentClientWrapper) Lookup(ctx context.Context, username string) (*userPkg.User, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lookup", "username", username)

	resp, err := h.hostServiceClient.Lookup(ctx, &proto.LookupRequest{
		Username: username,
	})

	if err != nil {
		st := status.Convert(err)
		for _, detail := range st.Details() {
			if protoUnknownUserError, ok := detail.(*proto.UnknownUserError); ok {
				return nil, userPkg.UnknownUserError(protoUnknownUserError.Username)
			}
		}
		return nil, unwrapGrpcStatusErrno(err)
	}

	return &userPkg.User{
		Uid:      resp.Uid,
		Gid:      resp.Gid,
		Username: resp.Username,
		Name:     resp.Name,
		HomeDir:  resp.Homedir,
	}, nil
}

func (h *AgentClientWrapper) LookupGroup(ctx context.Context, name string) (*userPkg.Group, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("LookupGroup", "group", name)

	resp, err := h.hostServiceClient.LookupGroup(ctx, &proto.LookupGroupRequest{
		Name: name,
	})
	if err != nil {
		st := status.Convert(err)
		for _, detail := range st.Details() {
			if protoUnknownGroupError, ok := detail.(*proto.UnknownGroupError); ok {
				return nil, userPkg.UnknownGroupError(protoUnknownGroupError.Name)
			}
		}
		return nil, unwrapGrpcStatusErrno(err)
	}

	return &userPkg.Group{
		Gid:  resp.Gid,
		Name: resp.Name,
	}, nil
}

func (h *AgentClientWrapper) Lstat(ctx context.Context, name string) (*types.Stat_t, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lstat", "name", name)

	resp, err := h.hostServiceClient.Lstat(ctx, &proto.LstatRequest{
		Name: name,
	})
	if err != nil {
		return nil, unwrapGrpcStatusErrno(err)
	}

	stat_t := types.Stat_t{
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
		Atim: types.Timespec{
			Sec:  resp.Atim.Sec,
			Nsec: resp.Atim.Nsec,
		},
		Mtim: types.Timespec{
			Sec:  resp.Mtim.Sec,
			Nsec: resp.Mtim.Nsec,
		},
		Ctim: types.Timespec{
			Sec:  resp.Ctim.Sec,
			Nsec: resp.Ctim.Nsec,
		},
	}

	return &stat_t, nil
}

func (h *AgentClientWrapper) ReadDir(ctx context.Context, name string) (<-chan types.DirEntResult, func()) {
	logger := log.MustLogger(ctx)

	logger.Debug("ReadDir", "name", name)

	ctx, cancel := context.WithCancel(ctx)

	dirEntResultCh := make(chan types.DirEntResult, 100)

	go func() {
		stream, err := h.hostServiceClient.ReadDir(ctx, &proto.ReadDirRequest{
			Name: name,
		})
		if err != nil {
			dirEntResultCh <- types.DirEntResult{Error: unwrapGrpcStatusErrno(err)}
			close(dirEntResultCh)
			return
		}

		for {
			dirEnt, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				dirEntResultCh <- types.DirEntResult{Error: unwrapGrpcStatusErrno(err)}
				close(dirEntResultCh)
				return
			}

			dirEntResultCh <- types.DirEntResult{
				DirEnt: types.DirEnt{
					Ino:  dirEnt.Ino,
					Type: uint8(dirEnt.Type),
					Name: dirEnt.Name,
				},
			}
		}

		close(dirEntResultCh)
	}()

	return dirEntResultCh, cancel
}

func (h *AgentClientWrapper) Mkdir(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Mkdir", "name", name)

	_, err := h.hostServiceClient.Mkdir(ctx, &proto.MkdirRequest{
		Name: name,
		Mode: mode,
	})
	if err != nil {
		return unwrapGrpcStatusErrno(err)
	}

	return nil
}

func (h *AgentClientWrapper) ReadFile(ctx context.Context, name string) (io.ReadCloser, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("ReadFile", "name", name)

	ctx, cancel := context.WithCancel(ctx)

	stream, err := h.hostServiceClient.ReadFile(ctx, &proto.ReadFileRequest{Name: name})
	if err != nil {
		cancel()
		return nil, unwrapGrpcStatusErrno(err)
	}

	// ReadFile will succeeds to create the stream before the server function is called.
	// Because of this, we require to read the first element of the stream here, as it
	// enables to catch the various errors we're expected to return.
	readFileResponse, err := stream.Recv()
	if err == io.EOF {
		cancel()
		return io.NopCloser(bytes.NewReader([]byte{})), nil
	}
	if err != nil {
		cancel()
		return nil, unwrapGrpcStatusErrno(err)
	}

	return &AgentClientWrapperReadFileReadCloser{
		Stream:     stream,
		CancelFunc: cancel,
		Data:       readFileResponse.Chunk,
	}, nil
}

func (h *AgentClientWrapper) Symlink(ctx context.Context, oldname, newname string) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Symlink", "oldname", oldname, "newname", newname)

	_, err := h.hostServiceClient.Symlink(ctx, &proto.SymlinkRequest{
		Oldname: oldname,
		Newname: newname,
	})

	if err != nil {
		return unwrapGrpcStatusErrno(err)
	}

	return nil
}

func (h *AgentClientWrapper) Readlink(ctx context.Context, name string) (string, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Readlink", "name", name)

	resp, err := h.hostServiceClient.ReadLink(ctx, &proto.ReadLinkRequest{
		Name: name,
	})

	if err != nil {
		return "", unwrapGrpcStatusErrno(err)
	}

	return resp.Destination, nil
}

func (h *AgentClientWrapper) Remove(ctx context.Context, name string) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Remove", "name", name)

	_, err := h.hostServiceClient.Remove(ctx, &proto.RemoveRequest{
		Name: name,
	})
	if err != nil {
		return unwrapGrpcStatusErrno(err)
	}

	return nil
}

func (h *AgentClientWrapper) Mknod(ctx context.Context, pathName string, mode uint32, dev uint64) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Mknod", "pathName", pathName, "mode", mode, "dev", dev)

	_, err := h.hostServiceClient.Mknod(ctx, &proto.MknodRequest{
		Path: pathName,
		Mode: mode,
		Dev:  dev,
	})
	if err != nil {
		return unwrapGrpcStatusErrno(err)
	}
	return nil
}

func (h *AgentClientWrapper) runStdinCopier(
	stdinReader io.Reader,
	stream grpc.BidiStreamingClient[proto.RunRequest, proto.RunResponse],
) error {
	buffer := make([]byte, 8196)
	for {
		n, err := stdinReader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return unwrapGrpcStatusErrno(err)
		}

		err = stream.Send(
			&proto.RunRequest{
				Data: &proto.RunRequest_StdinChunk{
					StdinChunk: buffer[:n],
				},
			},
		)
		if err != nil {
			return unwrapGrpcStatusErrno(err)
		}
	}
	return nil
}

func (h *AgentClientWrapper) Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Run", "cmd", cmd)

	stream, err := h.hostServiceClient.Run(ctx)
	if err != nil {
		return types.WaitStatus{}, unwrapGrpcStatusErrno(err)
	}

	err = stream.Send(
		&proto.RunRequest{
			Data: &proto.RunRequest_Cmd{
				Cmd: &proto.Cmd{
					Path:    cmd.Path,
					Args:    cmd.Args,
					EnvVars: cmd.Env,
					Dir:     cmd.Dir,
					Stdin:   cmd.Stdin != nil,
					Stdout:  cmd.Stdout != nil,
					Stderr:  cmd.Stderr != nil,
				},
			},
		},
	)
	if err != nil {
		return types.WaitStatus{}, errors.Join(
			unwrapGrpcStatusErrno(err),
			stream.CloseSend(),
		)
	}

	var wg sync.WaitGroup

	var stdinErr error
	if cmd.Stdin != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stdinErr = h.runStdinCopier(cmd.Stdin, stream)
		}()
	}

	var waitStatus types.WaitStatus
	var recvError error
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			recvError = unwrapGrpcStatusErrno(err)
			break
		}

		if respData, ok := resp.Data.(*proto.RunResponse_Waitstatus); ok {
			waitStatus.ExitCode = int(respData.Waitstatus.Exitcode)
			waitStatus.Exited = respData.Waitstatus.Exited
			waitStatus.Signal = respData.Waitstatus.Signal
			break
		} else if respData, ok := resp.Data.(*proto.RunResponse_StdoutChunk); ok {
			if cmd.Stdout == nil {
				panic("bug: received stdout chunk for nil stdout")
			}
			if _, err := cmd.Stdout.Write(respData.StdoutChunk); err != nil {
				recvError = unwrapGrpcStatusErrno(err)
				break
			}
		} else if respData, ok := resp.Data.(*proto.RunResponse_StderrChunk); ok {
			if cmd.Stderr == nil {
				panic("bug: received stderr chunk for nil stderr")
			}
			if _, err := cmd.Stderr.Write(respData.StderrChunk); err != nil {
				recvError = unwrapGrpcStatusErrno(err)
				break
			}
		} else {
			panic(fmt.Errorf("bug: unexpected response data: %#v", resp.Data))
		}
	}

	closeSendErr := stream.CloseSend()

	wg.Wait()

	err = errors.Join(
		stdinErr,
		recvError,
		closeSendErr,
	)
	if err != nil {
		return types.WaitStatus{}, err
	}

	return waitStatus, nil
}

func (h *AgentClientWrapper) WriteFile(ctx context.Context, name string, data io.Reader, perm uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("WriteFile", "name", name, "data", data, "perm", perm)

	stream, err := h.hostServiceClient.WriteFile(ctx)
	if err != nil {
		return unwrapGrpcStatusErrno(err)
	}

	err = stream.Send(
		&proto.WriteFileRequest{
			Data: &proto.WriteFileRequest_Metadata{
				Metadata: &proto.FileMetadata{
					Name: name,
					Perm: perm,
				},
			},
		},
	)
	if err != nil {
		_, closeAndRecvErr := stream.CloseAndRecv()
		return errors.Join(
			unwrapGrpcStatusErrno(err),
			closeAndRecvErr,
		)
	}

	var sendErr error
	buffer := make([]byte, 1024)
	for {
		n, err := data.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			sendErr = unwrapGrpcStatusErrno(err)
			break
		}

		err = stream.Send(
			&proto.WriteFileRequest{
				Data: &proto.WriteFileRequest_Chunk{
					Chunk: buffer[:n],
				},
			},
		)
		if err != nil {
			sendErr = unwrapGrpcStatusErrno(err)
			break
		}
	}

	_, closeAndRecvErr := stream.CloseAndRecv()
	return errors.Join(
		sendErr,
		unwrapGrpcStatusErrno(closeAndRecvErr),
	)
}

func (h *AgentClientWrapper) String() string {
	return h.BaseHost.String()
}

func (h *AgentClientWrapper) Type() string {
	return h.BaseHost.Type()
}

func (h *AgentClientWrapper) Close(ctx context.Context) error {

	_, shutdownErr := h.hostServiceClient.Shutdown(ctx, &proto.Empty{})

	var spawnErr error
	if shutdownErr == nil {
		spawnErr = <-h.spawnErrCh
	}

	grpcClientConnErr := h.grpcClientConn.Close()

	hostCloseErr := h.BaseHost.Close(ctx)

	return errors.Join(
		shutdownErr,
		grpcClientConnErr,
		spawnErr,
		hostCloseErr,
	)
}
