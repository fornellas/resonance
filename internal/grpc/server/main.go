package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"github.com/fornellas/resonance/internal/grpc/proto"
)

type commandServiceServer struct {
	proto.UnimplementedCommandServiceServer
}

func (s *commandServiceServer) Ping(ctx context.Context, req *proto.PingRequest) (*proto.PingResponse, error) {
	return &proto.PingResponse{Message: "Pong"}, nil
}

type fileServiceServer struct {
	proto.UnimplementedFileServiceServer
}

func (s *fileServiceServer) PutFile(ctx context.Context, req *proto.PutFileRequest) (*proto.PutFileResponse, error) {
	filePath := fmt.Sprintf("%c%s", os.PathSeparator, req.Name)
	err := os.WriteFile(filePath, req.Content, os.FileMode(req.Perm))
	if err != nil {
		return nil, err
	}

	return &proto.PutFileResponse{Status: "File saved successfully"}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Register both services
	proto.RegisterCommandServiceServer(grpcServer, &commandServiceServer{})
	proto.RegisterFileServiceServer(grpcServer, &fileServiceServer{})

	log.Printf("Server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
