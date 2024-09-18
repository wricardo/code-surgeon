package grpc

import (
	"context"
	"log"
	"os/exec"

	"connectrpc.com/connect"
	"github.com/wricardo/code-surgeon/api"
)

// ExecuteBash executes a bash command and returns the stdout, stderr, and exit code.
func (h *Handler) ExecuteBash(ctx context.Context, req *connect.Request[api.ExecuteBashRequest]) (*connect.Response[api.ExecuteBashResponse], error) {
	cmd := exec.Command("bash", "-c", req.Msg.Command)
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			log.Printf("Bash execution failed: %v\n", stderr)
			return &connect.Response[api.ExecuteBashResponse]{Msg: &api.ExecuteBashResponse{Stdout: "", Stderr: stderr, ExitCode: int32(exitErr.ExitCode())}}, nil
		}
		return nil, err
	}
	return &connect.Response[api.ExecuteBashResponse]{Msg: &api.ExecuteBashResponse{Stdout: string(stdout), Stderr: "", ExitCode: 0}}, nil
}
