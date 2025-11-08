// cmd/app/main.go
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpcServer "github.com/darksuei/cloud-native-ffmpeg/internal/grpc"
	pb "github.com/darksuei/cloud-native-ffmpeg/proto"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {
	_ = godotenv.Load()

	// Read port from env or use default
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	// Create listener
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", port, err)
	}

	// Create gRPC server
	server := grpc.NewServer()
	ffmpegService := grpcServer.NewFFmpegServer()
	pb.RegisterFFmpegProcessorServer(server, ffmpegService)

	// Context to handle graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run gRPC server in background
	go func() {
		log.Printf("🚀 gRPC server running on port: %s", port)
		if err := server.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutdown signal received, terminating server...")
	server.GracefulStop()
	log.Println("Server terminated...")
}
