// cmd/test/main.go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pb "github.com/darksuei/cloud-native-ffmpeg/proto"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 3 {
		fmt.Println("Usage: go run cmd/test/main.go <inputfile> <ffmpeg_args>")
		os.Exit(1)
	}

	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	fmt.Printf("Testing gRPC server on port %s \n", port)

	filePath := os.Args[1]
	ffmpegArgs := os.Args[2]

	conn, err := grpc.Dial(fmt.Sprintf("localhost:%s", port), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := pb.NewFFmpegProcessorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	stream, err := client.ProcessStream(ctx)
	if err != nil {
		log.Fatalf("failed to start stream: %v", err)
	}

	// Step 1: send initial message (ffmpeg args)
	if err := stream.Send(&pb.ProcessRequest{Args: ffmpegArgs}); err != nil {
		log.Fatalf("failed to send args: %v", err)
	}

	// Step 2: open and stream file
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	buf := make([]byte, 32*1024) // 32 KB chunks
	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("read error: %v", err)
		}

		if err := stream.Send(&pb.ProcessRequest{Chunk: buf[:n]}); err != nil {
			log.Fatalf("failed to send chunk: %v", err)
		}
	}

	// Step 3: signal EOF
	if err := stream.Send(&pb.ProcessRequest{Eof: true}); err != nil {
		log.Fatalf("failed to send EOF: %v", err)
	}

	// Step 4: receive ffmpeg output until done
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("recv error: %v", err)
		}

		if len(resp.Chunk) > 0 {
			fmt.Print(string(resp.Chunk))
		}

		if resp.Done {
			fmt.Println("\n✅ Done processing file")
			break
		}
	}

	fmt.Println("Stream closed successfully")
}
