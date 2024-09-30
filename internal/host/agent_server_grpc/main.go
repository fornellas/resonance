package main

import (
	"context"
	"os"

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
	mode := os.FileMode(req.Mode)
	if err := os.Chmod(name, mode); err != nil {
		return nil, err
	}

	return &proto.ChmodResponse{Status: "chmod successful"}, nil
}

func (s *HostService) Chown(ctx context.Context, req *proto.ChownRequest) (*proto.ChownResponse, error) {
	name := req.Name
	uid := int(req.Uid)
	gid := int(req.Gid)

	if err := os.Chown(name, uid, gid); err != nil {
		return nil, err
	}

	return &proto.ChownResponse{Status: "chown successful"}, nil
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
