package main

import (
	"context"
	"log"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	proto "github.com/harishb93/telemetry-pipeline/internal/proto"
)

func TestGRPCIntegration(t *testing.T) {
	// Connect to the gRPC server
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	client := proto.NewCollectorServiceClient(conn)

	// Test GetAllGPUIds
	resp, err := client.GetAllGPUIds(context.Background(), &proto.Empty{})
	if err != nil {
		t.Fatalf("GetAllGPUIds failed: %v", err)
	}

	log.Printf("Received GPU IDs: %v", resp.GpuIds)

	// Add more tests for other gRPC methods as needed
}
