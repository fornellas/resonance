package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/fornellas/resonance/internal/host/agent_server_grpc/proto"
	aNet "github.com/fornellas/resonance/internal/host/agent_server_http/net"
)

type HostService struct {
	proto.UnimplementedHostServiceServer
}

func (s *HostService) Ping(ctx context.Context, req *proto.PingRequest) (*proto.PingResponse, error) {
	return &proto.PingResponse{Message: "Pong"}, nil
}

func (s *HostService) Chmod(ctx context.Context, req *proto.ChmodRequest) (*proto.ChmodResponse, error) {
	name := req.Name
	mode := req.Mode

	if !filepath.IsAbs(name) {
		return nil, fmt.Errorf("path must be absolute: %s", name)
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
		return nil, fmt.Errorf("path must be absolute: %s", name)
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

func (s *HostService) Mkdir(ctx context.Context, req *proto.MkdirRequest) (*proto.Empty, error) {
	name := req.Name
	mode := req.Mode

	if !filepath.IsAbs(name) {
		return nil, fmt.Errorf("path must be absolute: %s", name)
	}

	if err := syscall.Mkdir(name, mode); err != nil {
		return nil, getGrpcError(err)
	}
	return nil, getGrpcError(syscall.Chmod(name, mode))
}

func getGrpcError(err error) error {
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


func main() {

	conn := aNet.Conn{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}

	pipeListener := aNet.NewListener(conn)

	grpcServer := grpc.NewServer()

	proto.RegisterHostServiceServer(grpcServer, &HostService{})

	if err := grpcServer.Serve(pipeListener); err != nil {
		panic(err)
	}
}
