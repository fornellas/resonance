package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"google.golang.org/grpc"

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
