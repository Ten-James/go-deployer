package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/ten-james/go-deploy-system/shared/proto"
	"google.golang.org/grpc"
)

const (
	uploadDir = "./uploads"
	port      = ":9999"
)

var apiKey string

func main() {
	flag.StringVar(&apiKey, "api-key", "", "API key for authentication")
	flag.Parse()

	if apiKey == "" {
		log.Fatal("API key is required. Use -api-key flag")
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Set larger message size limits (100MB)
	maxMsgSize := 100 * 1024 * 1024
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	)
	proto.RegisterDeploymentServiceServer(s, &deploymentServer{})

	fmt.Printf("gRPC Deployment server listening on port %s\n", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}