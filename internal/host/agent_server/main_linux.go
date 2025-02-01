package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/internal/host/agent_server/proto"
	"github.com/fornellas/resonance/internal/host/lib"
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

func (s *HostService) Chmod(ctx context.Context, req *proto.ChmodRequest) (*proto.ChmodResponse, error) {
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

	return &proto.ChmodResponse{Status: "chmod successful"}, nil
}

func (s *HostService) Chown(ctx context.Context, req *proto.ChownRequest) (*proto.ChownResponse, error) {
	name := req.Name
	uid := int(req.Uid)
	gid := int(req.Gid)

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "Chown",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	if err := syscall.Chown(name, uid, gid); err != nil {
		return nil, err
	}

	return &proto.ChownResponse{Status: "chown successful"}, nil
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

func (s *HostService) ReadDir(ctx context.Context, req *proto.ReadDirRequest) (*proto.ReadDirResponse, error) {
	name := req.Name

	if !filepath.IsAbs(name) {
		return nil, &fs.PathError{
			Op:   "ReadDir",
			Path: name,
			Err:  errors.New("path must be absolute"),
		}
	}

	dirEnts, err := lib.ReadDir(ctx, name)
	if err != nil {
		return nil, s.getGrpcError(err)
	}

	protoDirEnts := make([]*proto.DirEnt, 0, len(dirEnts))

	for _, dirEnt := range dirEnts {
		protoDirEnt := &proto.DirEnt{
			Name: dirEnt.Name,
			Type: int32(dirEnt.Type),
			Ino:  dirEnt.Ino,
		}
		protoDirEnts = append(protoDirEnts, protoDirEnt)
	}

	return &proto.ReadDirResponse{
		Entries: protoDirEnts,
	}, nil
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

func (s *HostService) Run(ctx context.Context, req *proto.RunRequest) (*proto.RunResponse, error) {
	var stdin io.Reader
	if req.Stdin != nil {
		stdin = bytes.NewReader(req.Stdin)
	}

	var stdout []byte
	stdoutBuff := bytes.NewBuffer(stdout)

	var stderr []byte
	stderrBuff := bytes.NewBuffer(stderr)

	cmd := host.Cmd{
		Path:   req.Path,
		Args:   req.Args,
		Env:    req.EnvVars,
		Dir:    req.Dir,
		Stdin:  stdin,
		Stdout: stdoutBuff,
		Stderr: stderrBuff,
	}

	waitStatus, err := lib.Run(ctx, cmd)
	if err != nil {
		return nil, s.getGrpcError(err)
	}

	return &proto.RunResponse{
		Waitstatus: &proto.WaitStatus{
			Exitcode: int64(waitStatus.ExitCode),
			Exited:   waitStatus.Exited,
			Signal:   waitStatus.Signal,
		},
		Stdout: stdoutBuff.Bytes(),
		Stderr: stderrBuff.Bytes(),
	}, nil
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
	ioConn := lib.IOConn{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}

	pipeListener := lib.NewListener(ioConn)

	grpcServer := grpc.NewServer()

	proto.RegisterHostServiceServer(grpcServer, &HostService{
		grpcServer: grpcServer,
	})

	if err := grpcServer.Serve(pipeListener); err != nil {
		panic(err)
	}
}
