package main

import (
	"context"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/fornellas/resonance/internal/host/agent_server_grpc/proto"
)

func main() {
	client, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not create client: %v", err)
	}
	defer client.Close()

	commandClient := proto.NewCommandServiceClient(client)
	fileClient := proto.NewFileServiceClient(client)

	// Ping
	pingResp, err := commandClient.Ping(context.Background(), &proto.PingRequest{})
	if err != nil {
		log.Fatalf("could not ping: %v", err)
	}
	log.Printf("Ping response: %s", pingResp.Message)

	// PutFile
	fileContent, err := os.ReadFile("/tmp/romo.txt")
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}
	perm := int32(0644)
	putFileResp, err := fileClient.PutFile(context.Background(), &proto.PutFileRequest{
		Name:    "/tmp/remo.txt",
		Content: fileContent,
		Perm:    perm,
	})
	if err != nil {
		log.Fatalf("could not put file: %v", err)
	}
	log.Printf("PutFile response: %s", putFileResp.Status)
}
