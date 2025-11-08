package grpc

import (
	"fmt"
	"io"

	"github.com/darksuei/cloud-native-ffmpeg/internal/ffmpeg"
	pb "github.com/darksuei/cloud-native-ffmpeg/proto"
)

// FFmpegServer implements the gRPC service defined in processor.proto
type FFmpegServer struct {
	pb.UnimplementedFFmpegProcessorServer
}

// NewFFmpegServer returns a new instance
func NewFFmpegServer() *FFmpegServer {
	return &FFmpegServer{}
}

// ProcessStream handles bidirectional streaming for ffmpeg processing.
func (s *FFmpegServer) ProcessStream(stream pb.FFmpegProcessor_ProcessStreamServer) error {
	ctx := stream.Context()

	// Step 1: receive initial message (ffmpeg args)
	req, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive initial request: %w", err)
	}

	runner, err := ffmpeg.NewRunner(ctx, req.Args)
	if err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}
	defer runner.CloseInput()

	// Step 2: stream ffmpeg output back to client
	go func() {
		runner.ReadOutput(ctx, func(chunk []byte) error {
			return stream.Send(&pb.ProcessResponse{Chunk: chunk})
		})
	}()

	// Step 3: handle incoming data from client
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error receiving stream data: %w", err)
		}

		if len(req.Chunk) > 0 {
			if err := runner.WriteInput(req.Chunk); err != nil {
				return fmt.Errorf("error writing to ffmpeg stdin: %w", err)
			}
		}

		if req.Eof {
			break
		}
	}

	// Step 4: signal ffmpeg input complete
	defer runner.CloseInput()

	// Step 5: wait for ffmpeg to finish
	if err := runner.Wait(); err != nil {
		fmt.Println("ffmpeg exited with error:", err)
	}

	// Step 6: signal completion to client
	return stream.Send(&pb.ProcessResponse{Done: true})
}
