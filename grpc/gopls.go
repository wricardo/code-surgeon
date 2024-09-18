package grpc

import (
	"context"
	"fmt"
	"log"
	"os/exec"

	"connectrpc.com/connect"
	"github.com/wricardo/code-surgeon/api"
)

// ExecuteGoplsImplementations executes the `gopls implementations` command for a given file, line, and column,
// and returns the result as a gRPC response.
func (h *Handler) ExecuteGoplsImplementations(ctx context.Context, req *connect.Request[api.ExecuteGoplsImplementationsRequest]) (*connect.Response[api.ExecuteGoplsImplementationsResponse], error) {
	command := fmt.Sprintf("gopls implementations %s:%d:%d", req.Msg.FilePath, req.Msg.Line, req.Msg.Column)
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
			log.Printf("Gopls implementations execution failed: %v\n", stderr)
			return &connect.Response[api.ExecuteGoplsImplementationsResponse]{Msg: &api.ExecuteGoplsImplementationsResponse{Output: "", Error: stderr}}, nil
		}
		return nil, err
	}
	return &connect.Response[api.ExecuteGoplsImplementationsResponse]{Msg: &api.ExecuteGoplsImplementationsResponse{Output: string(output), Error: ""}}, nil
}
