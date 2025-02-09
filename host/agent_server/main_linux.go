package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"sync"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/fornellas/resonance/host/agent_server/proto"
	"github.com/fornellas/resonance/host/lib"
	hostNet "github.com/fornellas/resonance/host/net"
	"github.com/fornellas/resonance/host/types"
)

type HostService struct {
	proto.UnimplementedHostServiceServer
	grpcServer *grpc.Server
}

func (s *HostService) getGrpcError(err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	if errors.Is(err, fs.ErrExist) {
		return status.Error(codes.AlreadyExists, err.Error())
	}
	if errors.Is(err, fs.ErrNotExist) {
		return status.Error(codes.NotFound, err.Error())
	}
	return err
}

func (s *HostService) Ping(ctx context.Context, req *proto.PingRequest) (*proto.PingResponse, error) {
	return &proto.PingResponse{Message: "Pong"}, nil
}

func (s *HostService) Geteuid(ctx context.Context, _ *proto.Empty) (*proto.GeteuidResponse, error) {
	return &proto.GeteuidResponse{
		Uid: uint64(syscall.Geteuid()),
	}, nil
}

func (s *HostService) Getegid(ctx context.Context, _ *proto.Empty) (*proto.GetegidResponse, error) {
	return &proto.GetegidResponse{
		Gid: uint64(syscall.Getegid()),
	}, nil
}

func (s *HostService) Chmod(ctx context.Context, req *proto.ChmodRequest) (*proto.Empty, error) {
	name := req.Name
	mode := req.Mode

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "Chmod",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	if err := syscall.Chmod(name, mode); err != nil {
		return nil, err
	}

	return &proto.Empty{}, nil
}

func (s *HostService) Lchown(ctx context.Context, req *proto.LchownRequest) (*proto.Empty, error) {
	name := req.Name
	uid := int(req.Uid)
	gid := int(req.Gid)

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "Lchown",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	if err := syscall.Lchown(name, uid, gid); err != nil {
		return nil, err
	}

	return &proto.Empty{}, nil
}

func (s *HostService) Lookup(ctx context.Context, req *proto.LookupRequest) (*proto.LookupResponse, error) {
	name := req.Username
	user, err := user.Lookup(name)
	if err != nil {
		return nil, err
	}

	return &proto.LookupResponse{
		Uid:      user.Uid,
		Gid:      user.Gid,
		Username: user.Username,
		Name:     user.Name,
		Homedir:  user.HomeDir,
	}, nil
}

func (s *HostService) LookupGroup(ctx context.Context, req *proto.LookupGroupRequest) (*proto.LookupGroupResponse, error) {
	name := req.Name
	group, err := user.LookupGroup(name)
	if err != nil {
		return nil, err
	}

	return &proto.LookupGroupResponse{
		Gid:  group.Gid,
		Name: group.Name,
	}, nil
}

func (s *HostService) Lstat(ctx context.Context, req *proto.LstatRequest) (*proto.LstatResponse, error) {
	name := req.Name

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "Lstat",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	var syscallStat_t syscall.Stat_t

	if err := syscall.Lstat(name, &syscallStat_t); err != nil {
		return nil, s.getGrpcError(err)
	}
	return &proto.LstatResponse{
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
		Atim: &proto.Timespec{
			Sec:  int64(syscallStat_t.Atim.Sec),
			Nsec: int64(syscallStat_t.Atim.Nsec),
		},
		Mtim: &proto.Timespec{
			Sec:  int64(syscallStat_t.Mtim.Sec),
			Nsec: int64(syscallStat_t.Mtim.Nsec),
		},
		Ctim: &proto.Timespec{
			Sec:  int64(syscallStat_t.Ctim.Sec),
			Nsec: int64(syscallStat_t.Ctim.Nsec),
		},
	}, nil
}

func (s *HostService) ReadDir(
	req *proto.ReadDirRequest,
	stream grpc.ServerStreamingServer[proto.DirEnt],
) error {
	name := req.Name

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "ReadDir",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	dirEntResultCh, cancel := lib.LocalReadDir(stream.Context(), name)
	defer cancel()

	for dirEntResult := range dirEntResultCh {
		if dirEntResult.Error != nil {
			return s.getGrpcError(dirEntResult.Error)
		}

		err := stream.Send(&proto.DirEnt{
			Name: dirEntResult.DirEnt.Name,
			Type: int32(dirEntResult.DirEnt.Type),
			Ino:  dirEntResult.DirEnt.Ino,
		})
		if err != nil {
			return s.getGrpcError(err)
		}
	}

	return nil
}

func (s *HostService) Mkdir(ctx context.Context, req *proto.MkdirRequest) (*proto.Empty, error) {
	name := req.Name
	mode := req.Mode

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "Mkdir",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	if err := syscall.Mkdir(name, mode); err != nil {
		return nil, s.getGrpcError(err)
	}
	return nil, s.getGrpcError(syscall.Chmod(name, mode))
}

func (s *HostService) ReadFile(
	req *proto.ReadFileRequest, stream grpc.ServerStreamingServer[proto.ReadFileResponse],
) error {
	name := req.Name

	if !filepath.IsAbs(name) {
		return &fs.PathError{
			Op:   "ReadFile",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	file, err := os.Open(name)

	if err != nil {
		return s.getGrpcError(err)
	}

	defer file.Close()

	buf := make([]byte, 8192)

	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return s.getGrpcError(err)
		}

		err = stream.Send(&proto.ReadFileResponse{
			Chunk: buf[:n],
		})
		if err != nil {
			return s.getGrpcError(err)
		}
	}

	return nil
}

func (s *HostService) Symlink(ctx context.Context, req *proto.SymlinkRequest) (*proto.Empty, error) {
	if !filepath.IsAbs(req.Newname) {
		return nil, &fs.PathError{
			Op:   "Symlink",
			Path: req.Newname,
			Err:  errors.New("path must be absolute"),
		}
	}

	return nil, s.getGrpcError(syscall.Symlink(req.Oldname, req.Newname))
}

func (s *HostService) ReadLink(ctx context.Context, req *proto.ReadLinkRequest) (*proto.ReadLinkResponse, error) {
	name := req.Name

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "ReadLink",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	destination, err := os.Readlink(name)
	if err != nil {
		return nil, s.getGrpcError(err)
	}

	return &proto.ReadLinkResponse{
		Destination: destination,
	}, nil
}

func (s *HostService) Remove(ctx context.Context, req *proto.RemoveRequest) (*proto.Empty, error) {
	name := req.Name

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "Remove",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	if err := os.Remove(name); err != nil {
		return nil, s.getGrpcError(err)
	}

	return nil, nil
}

func (s *HostService) Mknod(ctx context.Context, req *proto.MknodRequest) (*proto.Empty, error) {
	if !filepath.IsAbs(req.Path) {
		return nil, fmt.Errorf("path must be absolute: %s", req.Path)
	}

	if req.Dev != uint64(int(req.Dev)) {
		return nil, fmt.Errorf("dev value is too big: %#v", req.Dev)
	}

	if err := syscall.Mknod(req.Path, req.Mode, int(req.Dev)); err != nil {
		return nil, s.getGrpcError(err)
	}

	return nil, s.getGrpcError(syscall.Chmod(req.Path, req.Mode&07777))
}

func (s *HostService) runStdinCopier(
	stream grpc.BidiStreamingServer[proto.RunRequest, proto.RunResponse],
	stdinWriter io.Writer,
) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return s.getGrpcError(err)
		}
		stdinChunk, ok := req.Data.(*proto.RunRequest_StdinChunk)
		if !ok {
			panic(fmt.Sprintf("bug: unexpected request data: %#v", req.Data))
		}
		if _, err := stdinWriter.Write(stdinChunk.StdinChunk); err != nil {
			return s.getGrpcError(err)
		}
	}
	return nil
}

func (s *HostService) runReaderCopier(
	stream grpc.BidiStreamingServer[proto.RunRequest, proto.RunResponse],
	wg *sync.WaitGroup,
	streamErr *error,
	getRunResponse func([]byte) *proto.RunResponse,
) (
	writeCloser io.WriteCloser,
	err error,
) {
	var reader io.Reader
	reader, writeCloser, err = os.Pipe()
	if err != nil {
		return nil, s.getGrpcError(err)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		buffer := make([]byte, 8196)
		for {
			n, err := reader.Read(buffer)
			if err == io.EOF {
				break
			}
			if err != nil {
				*streamErr = s.getGrpcError(err)
			}

			if err = stream.Send(getRunResponse(buffer[:n])); err != nil {
				*streamErr = s.getGrpcError(err)
			}
		}
	}()
	return writeCloser, nil
}

func (s *HostService) runStartStreamGoroutines(
	cmd *proto.Cmd,
	stream grpc.BidiStreamingServer[proto.RunRequest, proto.RunResponse],
	stdinErr *error,
	stdoutErr *error,
	stderrErr *error,
) (
	stdinReader io.ReadCloser,
	stdoutWriter io.WriteCloser,
	stderrWriter io.WriteCloser,
	wg *sync.WaitGroup,
	err error,
) {
	wg = &sync.WaitGroup{}

	if cmd.Stdin {
		var stdinWriter io.Writer
		stdinReader, stdinWriter, err = os.Pipe()
		if err != nil {
			return nil, nil, nil, wg, s.getGrpcError(err)
		}
		go func() {
			*stdinErr = s.runStdinCopier(stream, stdinWriter)
		}()
	}

	if cmd.Stdout {
		stdoutWriter, err = s.runReaderCopier(
			stream, wg, stdoutErr,
			func(buffer []byte) *proto.RunResponse {
				return &proto.RunResponse{
					Data: &proto.RunResponse_StdoutChunk{
						StdoutChunk: buffer,
					},
				}
			},
		)
		if err != nil {
			return nil, nil, nil, wg, err
		}
	}

	if cmd.Stderr {
		stderrWriter, err = s.runReaderCopier(
			stream, wg, stderrErr,
			func(buffer []byte) *proto.RunResponse {
				return &proto.RunResponse{
					Data: &proto.RunResponse_StderrChunk{
						StderrChunk: buffer,
					},
				}
			},
		)
		if err != nil {
			return nil, nil, nil, wg, err
		}
	}

	return stdinReader, stdoutWriter, stderrWriter, wg, nil
}

func (s *HostService) Run(stream grpc.BidiStreamingServer[proto.RunRequest, proto.RunResponse]) error {
	req, runErr := stream.Recv()
	if runErr != nil {
		return s.getGrpcError(runErr)
	}
	data, ok := req.Data.(*proto.RunRequest_Cmd)
	if !ok {
		panic(fmt.Errorf("bug: unexpected request data: %#v", req.Data))
	}

	var stdinErr error
	var stdoutErr error
	var stderrErr error

	stdinReader, stdoutWriter, stderrWriter, wg, err := s.runStartStreamGoroutines(
		data.Cmd, stream, &stdinErr, &stdoutErr, &stderrErr,
	)
	if err != nil {
		return err
	}

	if len(data.Cmd.EnvVars) == 0 {
		data.Cmd.EnvVars = types.DefaultEnv
	}

	cmd := types.Cmd{
		Path:   data.Cmd.Path,
		Args:   data.Cmd.Args,
		Env:    data.Cmd.EnvVars,
		Dir:    data.Cmd.Dir,
		Stdin:  stdinReader,
		Stdout: stdoutWriter,
		Stderr: stderrWriter,
	}
	waitStatus, runErr := lib.LocalRun(stream.Context(), cmd)
	if runErr != nil {
		runErr = s.getGrpcError(runErr)
	}

	var stdinCloseErr error
	if data.Cmd.Stdin {
		stdinCloseErr = stdinReader.Close()
	}

	var stdoutCloseErr error
	if data.Cmd.Stdout {
		stdoutCloseErr = stdoutWriter.Close()
	}

	var stderrCloseErr error
	if data.Cmd.Stderr {
		stderrCloseErr = stderrWriter.Close()
	}

	wg.Wait()

	err = errors.Join(
		runErr,
		stdinErr,
		stdoutErr,
		stderrErr,
		stdinCloseErr,
		stdoutCloseErr,
		stderrCloseErr,
	)
	if err != nil {
		return err
	}

	err = stream.Send(&proto.RunResponse{
		Data: &proto.RunResponse_Waitstatus{
			Waitstatus: &proto.WaitStatus{
				Exitcode: int64(waitStatus.ExitCode),
				Exited:   waitStatus.Exited,
				Signal:   waitStatus.Signal,
			},
		},
	})
	if err != nil {
		return s.getGrpcError(err)
	}

	return nil
}

func (s *HostService) WriteFile(
	stream grpc.ClientStreamingServer[proto.WriteFileRequest, proto.Empty],
) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	writeFileRequest_Metadata, ok := req.Data.(*proto.WriteFileRequest_Metadata)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "first message must be 'metadata'")
	}
	metadata := writeFileRequest_Metadata.Metadata

	if !filepath.IsAbs(metadata.Name) {
		return &fs.PathError{
			Op:   "WriteFile",
			Path: metadata.Name,
			Err:  errors.New("path must be absolute"),
		}
	}

	file, err := os.OpenFile(
		metadata.Name,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		os.FileMode(metadata.Perm),
	)
	if err != nil {
		return s.getGrpcError(err)
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Join(
				s.getGrpcError(err),
				s.getGrpcError(file.Close()),
			)
		}

		chunk, ok := req.Data.(*proto.WriteFileRequest_Chunk)
		if !ok {
			return errors.Join(
				status.Errorf(codes.InvalidArgument, "second message onwards must be 'chunk'"),
				s.getGrpcError(file.Close()),
			)
		}

		if _, err := file.Write(chunk.Chunk); err != nil {
			return errors.Join(
				s.getGrpcError(err),
				s.getGrpcError(file.Close()),
			)
		}
	}

	if err := file.Close(); err != nil {
		return s.getGrpcError(err)
	}

	if err = syscall.Chmod(metadata.Name, metadata.Perm); err != nil {
		return s.getGrpcError(err)
	}

	if err := stream.SendAndClose(&proto.Empty{}); err != nil {
		return s.getGrpcError(err)
	}

	return nil
}

func (s *HostService) Shutdown(ctx context.Context, _ *proto.Empty) (*proto.Empty, error) {
	go func() {
		// When GracefulStop() is executed, it'll close the connection (and the pipe), generating
		// a SIGPIPE, so we need to ignore the signal
		signal.Ignore(syscall.SIGPIPE)
		s.grpcServer.GracefulStop()
	}()
	return nil, nil
}

func main() {
	ioConn := hostNet.IOConn{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}

	pipeListener := hostNet.NewListener(ioConn)

	grpcServer := grpc.NewServer()

	proto.RegisterHostServiceServer(grpcServer, &HostService{
		grpcServer: grpcServer,
	})

	if err := grpcServer.Serve(pipeListener); err != nil {
		panic(err)
	}
}
