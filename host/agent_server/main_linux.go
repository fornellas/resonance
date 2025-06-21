package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	userPkg "os/user"
	"path/filepath"
	"strconv"
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

var errnoTogRPCCode = map[syscall.Errno]codes.Code{
	syscall.E2BIG:  codes.OutOfRange,
	syscall.EACCES: codes.PermissionDenied,
	// syscall.EADDRINUSE:	codes.,
	// syscall.EADDRNOTAVAIL:	codes.,
	syscall.EAFNOSUPPORT: codes.InvalidArgument,
	syscall.EAGAIN:       codes.Unavailable,
	// syscall.EALREADY:	codes.,
	// syscall.EBADE:	codes.,
	// syscall.EBADF:	codes.,
	// syscall.EBADFD:	codes.,
	syscall.EBADMSG: codes.InvalidArgument,
	// syscall.EBADR:	codes.,
	syscall.EBADRQC: codes.InvalidArgument,
	// syscall.c:	codes.,
	// syscall.EBUSY:	codes.,
	syscall.ECANCELED: codes.Canceled,
	// syscall.ECHILD:	codes.,
	// syscall.ECHRNG:	codes.,
	// syscall.ECOMM:	codes.,
	syscall.ECONNABORTED: codes.Aborted,
	// syscall.ECONNREFUSED:	codes.,
	// syscall.ECONNRESET:	codes.,
	// syscall.EDEADLK:	codes.,
	// syscall.EDEADLOCK:	codes.,
	syscall.EDESTADDRREQ: codes.InvalidArgument,
	syscall.EDOM:         codes.InvalidArgument,
	// syscall.EDQUOT:	codes.,
	syscall.EEXIST: codes.AlreadyExists,
	// syscall.EFAULT:	codes.,
	syscall.EFBIG: codes.OutOfRange,
	// syscall.EHOSTDOWN:	codes.,
	// syscall.EHOSTUNREACH:	codes.,
	// syscall.EHWPOISON:	codes.,
	// syscall.EIDRM:	codes.,
	syscall.EILSEQ: codes.InvalidArgument,
	// syscall.EINPROGRESS:	codes.,
	// syscall.EINTR:	codes.,
	syscall.EINVAL: codes.InvalidArgument,
	// syscall.EIO:	codes.,
	// syscall.EISCONN:	codes.,
	// syscall.EISDIR:	codes.,
	// syscall.EISNAM:	codes.,
	// syscall.EKEYEXPIRED:	codes.,
	// syscall.EKEYREJECTED:	codes.,
	// syscall.EKEYREVOKED:	codes.,
	// syscall.EL2HLT:	codes.,
	// syscall.EL2NSYNC:	codes.,
	// syscall.EL3HLT:	codes.,
	// syscall.EL3RST:	codes.,
	// syscall.ELIBACC:	codes.,
	// syscall.ELIBBAD:	codes.,
	// syscall.ELIBMAX:	codes.,
	// syscall.ELIBSCN:	codes.,
	// syscall.ELIBEXEC:	codes.,
	// syscall.ELNRNG:	codes.,
	// syscall.ELOOP:	codes.,
	// syscall.EMEDIUMTYPE:	codes.,
	syscall.EMFILE:   codes.ResourceExhausted,
	syscall.EMLINK:   codes.ResourceExhausted,
	syscall.EMSGSIZE: codes.OutOfRange,
	// syscall.EMULTIHOP:	codes.,
	syscall.ENAMETOOLONG: codes.OutOfRange,
	// syscall.ENETDOWN:	codes.,
	syscall.ENETRESET: codes.Aborted,
	// syscall.ENETUNREACH:	codes.,
	// syscall.ENFILE:	codes.,
	// syscall.ENOANO:	codes.,
	// syscall.ENOBUFS:	codes.,
	// syscall.ENODATA:	codes.,
	// syscall.ENODEV:	codes.,
	syscall.ENOENT: codes.NotFound,
	// syscall.ENOEXEC:	codes.,
	// syscall.ENOKEY:	codes.,
	// syscall.ENOLCK:	codes.,
	// syscall.ENOLINK:	codes.,
	// syscall.ENOMEDIUM:	codes.,
	// syscall.ENOMEM:	codes.,
	// syscall.ENOMSG:	codes.,
	// syscall.ENONET:	codes.,
	// syscall.ENOPKG:	codes.,
	// syscall.ENOPROTOOPT:	codes.,
	// syscall.ENOSPC:	codes.,
	// syscall.ENOSR:	codes.,
	// syscall.ENOSTR:	codes.,
	// syscall.ENOSYS:	codes.,
	// syscall.ENOTBLK:	codes.,
	// syscall.ENOTCONN:	codes.,
	// syscall.ENOTDIR:	codes.,
	syscall.ENOTEMPTY: codes.FailedPrecondition,
	// syscall.ENOTRECOVERABLE:	codes.,
	// syscall.ENOTSOCK:	codes.,
	// syscall.ENOTSUP:	codes.,
	// syscall.ENOTTY:	codes.,
	// syscall.ENOTUNIQ:	codes.,
	// syscall.ENXIO:	codes.,
	// syscall.EOPNOTSUPP:	codes.,
	syscall.EOVERFLOW: codes.OutOfRange,
	// syscall.EOWNERDEAD:	codes.,
	syscall.EPERM: codes.PermissionDenied,
	// syscall.EPFNOSUPPORT:	codes.,
	// syscall.EPIPE:	codes.,
	// syscall.EPROTO:	codes.,
	// syscall.EPROTONOSUPPORT:	codes.,
	// syscall.EPROTOTYPE:	codes.,
	// syscall.ERANGE:	codes.,
	// syscall.EREMCHG:	codes.,
	// syscall.EREMOTE:	codes.,
	// syscall.EREMOTEIO:	codes.,
	// syscall.ERESTART:	codes.,
	// syscall.ERFKILL:	codes.,
	// syscall.EROFS:	codes.,
	// syscall.ESHUTDOWN:	codes.,
	syscall.ESPIPE:          codes.OutOfRange,
	syscall.ESOCKTNOSUPPORT: codes.InvalidArgument,
	// syscall.ESRCH:	codes.,
	// syscall.ESTALE:	codes.,
	// syscall.ESTRPIPE:	codes.,
	syscall.ETIME:     codes.DeadlineExceeded,
	syscall.ETIMEDOUT: codes.DeadlineExceeded,
	// syscall.ETOOMANYREFS:	codes.,
	// syscall.ETXTBSY:	codes.,
	// syscall.EUCLEAN:	codes.,
	// syscall.EUNATCH:	codes.,
	syscall.EUSERS: codes.ResourceExhausted,
	// syscall.EWOULDBLOCK:	codes.,
	// syscall.EXDEV:	codes.,
	syscall.EXFULL: codes.ResourceExhausted,
}

type HostService struct {
	proto.UnimplementedHostServiceServer
	grpcServer *grpc.Server
}

func (s *HostService) getGrpcStatusErrnoErr(err error) error {
	if err == nil {
		return nil
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		var st *status.Status
		if code, ok := errnoTogRPCCode[errno]; ok {
			st = status.New(code, err.Error())
		} else {
			st = status.New(codes.Unknown, err.Error())
		}
		ds, err := st.WithDetails(&proto.Errno{Errno: uint64(errno)})
		if err != nil {
			return st.Err()
		}
		return ds.Err()
	}
	return status.New(codes.Unknown, err.Error()).Err()
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
		return nil, status.Errorf(codes.InvalidArgument, "path must be absolute")
	}

	if err := syscall.Chmod(name, mode); err != nil {
		return nil, s.getGrpcStatusErrnoErr(err)
	}

	return &proto.Empty{}, nil
}

func (s *HostService) Lchown(ctx context.Context, req *proto.LchownRequest) (*proto.Empty, error) {
	name := req.Name
	uid := int(req.Uid)
	gid := int(req.Gid)

	if !filepath.IsAbs(name) {
		return nil, status.Errorf(codes.InvalidArgument, "path must be absolute")
	}

	if err := syscall.Lchown(name, uid, gid); err != nil {
		return nil, s.getGrpcStatusErrnoErr(err)
	}

	return &proto.Empty{}, nil
}

func (s *HostService) Lookup(ctx context.Context, req *proto.LookupRequest) (*proto.LookupResponse, error) {
	user, err := userPkg.Lookup(req.Username)
	if err != nil {
		var unknownUserError userPkg.UnknownUserError
		if errors.As(err, &unknownUserError) {
			st := status.New(codes.NotFound, err.Error())
			ds, err := st.WithDetails(&proto.UnknownUserError{Username: req.Username})
			if err != nil {
				return nil, st.Err()
			}
			return nil, ds.Err()
		}
		return nil, s.getGrpcStatusErrnoErr(err)
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
	group, err := userPkg.LookupGroup(req.Name)
	if err != nil {
		var unknownGroupError userPkg.UnknownGroupError
		if errors.As(err, &unknownGroupError) {
			st := status.New(codes.NotFound, err.Error())
			ds, err := st.WithDetails(&proto.UnknownGroupError{Name: req.Name})
			if err != nil {
				return nil, st.Err()
			}
			return nil, ds.Err()
		}
		return nil, s.getGrpcStatusErrnoErr(err)
	}

	return &proto.LookupGroupResponse{
		Gid:  group.Gid,
		Name: group.Name,
	}, nil
}

func (s *HostService) Lstat(ctx context.Context, req *proto.LstatRequest) (*proto.LstatResponse, error) {
	name := req.Name

	if !filepath.IsAbs(name) {
		return nil, status.Errorf(codes.InvalidArgument, "path must be absolute")
	}

	var syscallStat_t syscall.Stat_t

	if err := syscall.Lstat(name, &syscallStat_t); err != nil {
		return nil, s.getGrpcStatusErrnoErr(err)
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
		return status.Errorf(codes.InvalidArgument, "path must be absolute")
	}

	dirEntResultCh, cancel := lib.LocalReadDir(stream.Context(), name)
	defer cancel()

	for dirEntResult := range dirEntResultCh {
		if dirEntResult.Error != nil {
			return s.getGrpcStatusErrnoErr(dirEntResult.Error)
		}

		err := stream.Send(&proto.DirEnt{
			Name: dirEntResult.DirEnt.Name,
			Type: int32(dirEntResult.DirEnt.Type),
			Ino:  dirEntResult.DirEnt.Ino,
		})
		if err != nil {
			return s.getGrpcStatusErrnoErr(err)
		}
	}

	return nil
}

func (s *HostService) Mkdir(ctx context.Context, req *proto.MkdirRequest) (*proto.Empty, error) {
	name := req.Name
	mode := req.Mode

	if !filepath.IsAbs(name) {
		return nil, status.Errorf(codes.InvalidArgument, "path must be absolute")
	}

	if err := syscall.Mkdir(name, mode); err != nil {
		return nil, s.getGrpcStatusErrnoErr(err)
	}
	return nil, s.getGrpcStatusErrnoErr(syscall.Chmod(name, mode))
}

func (s *HostService) ReadFile(
	req *proto.ReadFileRequest, stream grpc.ServerStreamingServer[proto.ReadFileResponse],
) (retErr error) {
	name := req.Name

	if !filepath.IsAbs(name) {
		return status.Errorf(codes.InvalidArgument, "path must be absolute")
	}

	file, err := os.Open(name)

	if err != nil {
		return s.getGrpcStatusErrnoErr(err)
	}

	defer func() { retErr = errors.Join(retErr, file.Close()) }()

	buf := make([]byte, 8192)
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			return status.Error(codes.Canceled, ctx.Err().Error())
		default:
			n, err := file.Read(buf)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return s.getGrpcStatusErrnoErr(err)
			}

			err = stream.Send(&proto.ReadFileResponse{
				Chunk: buf[:n],
			})
			if err != nil {
				return s.getGrpcStatusErrnoErr(err)
			}
		}
	}
}

func (s *HostService) Symlink(ctx context.Context, req *proto.SymlinkRequest) (*proto.Empty, error) {
	if !filepath.IsAbs(req.Newname) {
		return nil, status.Errorf(codes.InvalidArgument, "path must be absolute")
	}

	return nil, s.getGrpcStatusErrnoErr(syscall.Symlink(req.Oldname, req.Newname))
}

func (s *HostService) ReadLink(ctx context.Context, req *proto.ReadLinkRequest) (*proto.ReadLinkResponse, error) {
	name := req.Name

	if !filepath.IsAbs(name) {
		return nil, status.Errorf(codes.InvalidArgument, "path must be absolute")
	}

	destination, err := os.Readlink(name)
	if err != nil {
		return nil, s.getGrpcStatusErrnoErr(err)
	}

	return &proto.ReadLinkResponse{
		Destination: destination,
	}, nil
}

func (s *HostService) Remove(ctx context.Context, req *proto.RemoveRequest) (*proto.Empty, error) {
	name := req.Name

	if !filepath.IsAbs(name) {
		return nil, status.Errorf(codes.InvalidArgument, "path must be absolute")
	}

	if err := os.Remove(name); err != nil {
		return nil, s.getGrpcStatusErrnoErr(err)
	}

	return nil, nil
}

func (s *HostService) Mknod(ctx context.Context, req *proto.MknodRequest) (*proto.Empty, error) {
	if !filepath.IsAbs(req.Path) {
		return nil, status.Errorf(codes.InvalidArgument, "path must be absolute")
	}

	if req.Dev != uint64(int(req.Dev)) {
		return nil, status.Errorf(codes.InvalidArgument, "dev value is too big: %#v", req.Dev)
	}

	if err := syscall.Mknod(req.Path, req.Mode, int(req.Dev)); err != nil {
		return nil, s.getGrpcStatusErrnoErr(err)
	}

	return nil, s.getGrpcStatusErrnoErr(syscall.Chmod(req.Path, req.Mode&uint32(types.FileModeBitsMask)))
}

func (s *HostService) runStdinCopier(
	stream grpc.BidiStreamingServer[proto.RunRequest, proto.RunResponse],
	stdinWrieCloser io.WriteCloser,
) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return s.getGrpcStatusErrnoErr(err)
		}
		stdinChunk, ok := req.Data.(*proto.RunRequest_StdinChunk)
		if !ok {
			panic(fmt.Sprintf("bug: unexpected request data: %#v", req.Data))
		}

		if len(stdinChunk.StdinChunk) == 0 {
			return stdinWrieCloser.Close()
		}

		if _, err := stdinWrieCloser.Write(stdinChunk.StdinChunk); err != nil {
			return s.getGrpcStatusErrnoErr(err)
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
		return nil, s.getGrpcStatusErrnoErr(err)
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
				*streamErr = s.getGrpcStatusErrnoErr(err)
			}

			if err = stream.Send(getRunResponse(buffer[:n])); err != nil {
				*streamErr = s.getGrpcStatusErrnoErr(err)
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
		var stdinWriteCloser io.WriteCloser
		stdinReader, stdinWriteCloser, err = os.Pipe()
		if err != nil {
			return nil, nil, nil, wg, s.getGrpcStatusErrnoErr(err)
		}
		go func() {
			*stdinErr = s.runStdinCopier(stream, stdinWriteCloser)
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
			return nil, nil, nil, wg, s.getGrpcStatusErrnoErr(err)
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
			return nil, nil, nil, wg, s.getGrpcStatusErrnoErr(err)
		}
	}

	return stdinReader, stdoutWriter, stderrWriter, wg, nil
}

func (s *HostService) Run(stream grpc.BidiStreamingServer[proto.RunRequest, proto.RunResponse]) error {
	req, runErr := stream.Recv()
	if runErr != nil {
		return s.getGrpcStatusErrnoErr(runErr)
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
		return s.getGrpcStatusErrnoErr(err)
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
	runErr = s.getGrpcStatusErrnoErr(runErr)

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
				Exitcode: waitStatus.ExitCode,
				Exited:   waitStatus.Exited,
				Signal:   waitStatus.Signal,
			},
		},
	})
	if err != nil {
		return s.getGrpcStatusErrnoErr(err)
	}

	return nil
}

func (s *HostService) WriteFile(
	stream grpc.ClientStreamingServer[proto.WriteFileRequest, proto.Empty],
) error {
	req, err := stream.Recv()
	if err != nil {
		return s.getGrpcStatusErrnoErr(err)
	}

	writeFileRequest_Metadata, ok := req.Data.(*proto.WriteFileRequest_Metadata)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "first message must be 'metadata'")
	}
	metadata := writeFileRequest_Metadata.Metadata

	if !filepath.IsAbs(metadata.Name) {
		return status.Errorf(codes.InvalidArgument, "path must be absolute")
	}

	file, err := os.OpenFile(
		metadata.Name,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		os.FileMode(metadata.Perm),
	)
	if err != nil {
		return s.getGrpcStatusErrnoErr(err)
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Join(
				s.getGrpcStatusErrnoErr(err),
				s.getGrpcStatusErrnoErr(file.Close()),
			)
		}

		chunk, ok := req.Data.(*proto.WriteFileRequest_Chunk)
		if !ok {
			return errors.Join(
				status.Errorf(codes.InvalidArgument, "second message onwards must be 'chunk'"),
				s.getGrpcStatusErrnoErr(file.Close()),
			)
		}

		if _, err := file.Write(chunk.Chunk); err != nil {
			return errors.Join(
				s.getGrpcStatusErrnoErr(err),
				s.getGrpcStatusErrnoErr(file.Close()),
			)
		}
	}

	if err := file.Close(); err != nil {
		return s.getGrpcStatusErrnoErr(err)
	}

	if err = syscall.Chmod(metadata.Name, metadata.Perm); err != nil {
		return s.getGrpcStatusErrnoErr(err)
	}

	if err := stream.SendAndClose(&proto.Empty{}); err != nil {
		return s.getGrpcStatusErrnoErr(err)
	}

	return nil
}

func (s *HostService) AppendFile(
	stream grpc.ClientStreamingServer[proto.AppendFileRequest, proto.Empty],
) error {
	req, err := stream.Recv()
	if err != nil {
		return s.getGrpcStatusErrnoErr(err)
	}

	appendFileRequest_Metadata, ok := req.Data.(*proto.AppendFileRequest_Metadata)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "first message must be 'metadata'")
	}
	metadata := appendFileRequest_Metadata.Metadata

	if !filepath.IsAbs(metadata.Name) {
		return status.Errorf(codes.InvalidArgument, "path must be absolute")
	}

	file, err := os.OpenFile(
		metadata.Name,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		os.FileMode(metadata.Perm),
	)
	if err != nil {
		return s.getGrpcStatusErrnoErr(err)
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Join(
				s.getGrpcStatusErrnoErr(err),
				s.getGrpcStatusErrnoErr(file.Close()),
			)
		}

		chunk, ok := req.Data.(*proto.AppendFileRequest_Chunk)
		if !ok {
			return errors.Join(
				status.Errorf(codes.InvalidArgument, "second message onwards must be 'chunk'"),
				s.getGrpcStatusErrnoErr(file.Close()),
			)
		}

		if _, err := file.Write(chunk.Chunk); err != nil {
			return errors.Join(
				s.getGrpcStatusErrnoErr(err),
				s.getGrpcStatusErrnoErr(file.Close()),
			)
		}
	}

	if err := file.Close(); err != nil {
		return s.getGrpcStatusErrnoErr(err)
	}

	if err = syscall.Chmod(metadata.Name, metadata.Perm); err != nil {
		return s.getGrpcStatusErrnoErr(err)
	}

	if err := stream.SendAndClose(&proto.Empty{}); err != nil {
		return s.getGrpcStatusErrnoErr(err)
	}

	return nil
}

func stop() {

	procDirEntries, err := os.ReadDir("/proc")
	if err != nil {
		panic(err)
	}

	selfPid := os.Getpid()
	selfAbsPath, err := filepath.Abs(filepath.Base(os.Args[0]))
	if err != nil {
		panic(err)
	}
	selfCmdline := selfAbsPath + "\x00"

	for _, procDirEntry := range procDirEntries {
		if !procDirEntry.IsDir() {
			continue
		}

		dirName := procDirEntry.Name()
		var pid int
		if pid, err = strconv.Atoi(dirName); err != nil {
			continue
		}
		if pid == selfPid {
			continue
		}

		cmdlinePath := filepath.Join("/proc", dirName, "cmdline")
		cmdlineBytes, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue
		}

		if selfCmdline == string(cmdlineBytes) {
			if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
				panic(err)
			}
		}
	}
}

func main() {
	doStop := len(os.Args) > 1 && os.Args[1] == "--stop"

	defer func() {
		if !doStop {
			return
		}
		if err := os.Remove(os.Args[0]); err != nil {
			panic(err)
		}
	}()

	if doStop {
		stop()
		return
	}

	pipeListener := NewListener(hostNet.IOConn{
		Reader: os.Stdin,
		Writer: os.Stdout,
	})

	grpcServer := grpc.NewServer()

	proto.RegisterHostServiceServer(grpcServer, &HostService{
		grpcServer: grpcServer,
	})

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGPIPE)
	go func() {
		<-signalCh
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(pipeListener); err != nil {
		panic(err)
	}
}
