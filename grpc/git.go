package grpc

import (
	"context"
	"os/exec"
	"strings"

	"connectrpc.com/connect"
	"github.com/wricardo/code-surgeon/api"
)

func (h *Handler) GitDiff(ctx context.Context, req *connect.Request[api.GitDiffRequest]) (*connect.Response[api.GitDiffResponse], error) {
	cmd := exec.Command("git", "diff")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	response := &api.GitDiffResponse{
		Output: string(strings.ToValidUTF8(string(output), "")),
	}
	return &connect.Response[api.GitDiffResponse]{Msg: response}, nil
}

// func (h *Handler) gitDiffVersion2NotWorking(ctx context.Context, req *connect.Request[api.GitDiffRequest]) (*connect.Response[api.GitDiffResponse], error) {
// 	cmd := exec.Command("git", "status", "--porcelain")
// 	output, err := cmd.Output()
// 	if err != nil {
// 		return nil, err
// 	}
// 	lines := strings.Split(string(output), "\n")
// 	var modifiedFiles []string // ListAllModifications handles the ListAllModifications gRPC method

// 	for _, line := range lines {
// 		if len(line) > 0 {
// 			fields := strings.Fields(line)
// 			if len(fields) > 1 {
// 				if strings.HasSuffix(fields[1], ".pb.go") {
// 					fmt.Println("Skipping proto file: ", fields[1])
// 					continue
// 				}
// 				fmt.Println("Adding file: ", fields[1])

// 				modifiedFiles = append(modifiedFiles, fields[1])
// 			}
// 		}
// 	}
// 	response := &api.GitDiffResponse{}
// 	if req.Msg.ReturnContent {
// 		for _, filename := range modifiedFiles {
// 			content, err := readFileContent(filename)
// 			if err != nil {
// 				log.Printf("Warn: Error reading file content: %v\n", err)
// 				continue
// 			}
// 			response.Files = append(response.Files, &api.FileContent{Filename: filename, Content: content})
// 		}
// 	} else {
// 		for _, filename := range modifiedFiles {
// 			response.Files = append(response.Files, &api.FileContent{Filename: filename, Content: ""})
// 		}
// 	}
// 	return &connect.Response[api.GitDiffResponse]{Msg: response}, nil
// }

// // readFileContent reads the content of a file and returns it as a string
// func readFileContent(filename string) (string, error) {
// 	content, err := os.ReadFile(filename)
// 	if err != nil {
// 		return "", err
// 	}
// 	return string(content), nil
// }
