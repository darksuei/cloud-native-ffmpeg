package ffmpeg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
)

// Runner encapsulates an ffmpeg process.
type Runner struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// NewRunner creates and starts an ffmpeg process with the given args.
func NewRunner(ctx context.Context, argStr string) (*Runner, error) {
	args := parseArgs(argStr)
	fmt.Println("Spinning up a new ffmpeg process with args: ", args)

	// Instead of spinning up a new process, use an open slot in the pool.

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	return &Runner{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}, nil
}

// WriteInput writes raw input data to ffmpeg stdin.
func (r *Runner) WriteInput(chunk []byte) error {
	if len(chunk) == 0 {
		return nil
	}
	_, err := r.stdin.Write(chunk)
	return err
}

// CloseInput closes ffmpeg's stdin (signals EOF).
func (r *Runner) CloseInput() {
    if r.stdin != nil {
        _ = r.stdin.Close()
    }
    if r.stdout != nil {
        _ = r.stdout.Close()
    }
    if r.stderr != nil {
        _ = r.stderr.Close()
    }
}

// ReadOutput continuously reads stdout and calls the provided handler for each chunk.
func (r *Runner) ReadOutput(ctx context.Context, handler func([]byte) error) {
	reader := bufio.NewReader(r.stdout)
	buf := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := reader.Read(buf)
			if n > 0 {
				if handleErr := handler(buf[:n]); handleErr != nil {
					fmt.Println("handler error:", handleErr)
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					fmt.Println("error reading stdout:", err)
				}
				return
			}
		}
	}
}

// Wait waits for ffmpeg to finish.
func (r *Runner) Wait() error {
	return r.cmd.Wait()
}

// parseArgs splits a command-line string into args respecting quotes.
func parseArgs(argStr string) []string {
	if argStr == "" {
		return []string{}
	}
	return splitQuoted(argStr)
}

// splitQuoted — handles quoted arguments like "-vf 'scale=320:240'"
func splitQuoted(s string) []string {
	var args []string
	var current string
	inQuote := false

	for _, c := range s {
		switch c {
		case '\'':
			inQuote = !inQuote
		case ' ':
			if inQuote {
				current += string(c)
			} else if current != "" {
				args = append(args, current)
				current = ""
			}
		default:
			current += string(c)
		}
	}
	if current != "" {
		args = append(args, current)
	}
	return args
}
