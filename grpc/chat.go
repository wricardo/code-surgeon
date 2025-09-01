package grpc

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"

	"connectrpc.com/connect"
	"github.com/wricardo/code-surgeon/api"
	// Import the neo4j2 package
)

func (h *Handler) NewChat(ctx context.Context, req *connect.Request[api.NewChatRequest]) (*connect.Response[api.NewChatResponse], error) {
	// TODO: Re-implement without chatcli dependency
	return nil, errors.New("NewChat not implemented")
}

func (h *Handler) SendMessage(ctx context.Context, req *connect.Request[api.SendMessageRequest]) (*connect.Response[api.SendMessageResponse], error) {
	// TODO: Re-implement without chatcli dependency
	return nil, errors.New("SendMessage not implemented")
}

func (h *Handler) GetChat(ctx context.Context, req *connect.Request[api.GetChatRequest]) (*connect.Response[api.GetChatResponse], error) {
	// TODO: Re-implement without chatcli dependency
	return nil, errors.New("GetChat not implemented")
}

/*
// ParseCodebase handles the ParseCodebase gRPC method
func (*Handler) ParseCodebase(ctx context.Context, req *connect.Request[api.ParseCodebaseRequest]) (*connect.Response[api.ParseCodebaseResponse], error) {
	// Extract the file or directory path from the request
	fileOrDirectory := req.Msg.FileOrDirectory
	if fileOrDirectory == "" {
		fileOrDirectory = "." // Default to current directory if not provided
	}

	// Call the ParseDirectory function to parse the codebase
	parsedInfo, err := codesurgeon.ParseDirectory(fileOrDirectory)
	if err != nil {
		log.Printf("Error parsing directory: %v", err)
		return &connect.Response[api.ParseCodebaseResponse]{
			Msg: &api.ParseCodebaseResponse{},
		}, err
	}

	// Convert the parsed information to the API response format
	response := &api.ParseCodebaseResponse{
		Packages: convertParsedInfoToProto(parsedInfo),
	}

	// Return the response
	return &connect.Response[api.ParseCodebaseResponse]{Msg: response}, nil
}

func (h *Handler) Introduction(ctx context.Context, req *connect.Request[api.IntroductionRequest]) (*connect.Response[api.IntroductionResponse], error) {
	res, err := h.GetOpenAPI(ctx, connect.NewRequest(&api.GetOpenAPIRequest{}))
	if err != nil {
		return nil, err
	}

	intro, err := ai.GetGPTIntroduction(res.Msg.Openapi)
	if err != nil {
		return nil, err
	}

	return &connect.Response[api.IntroductionResponse]{
		Msg: &api.IntroductionResponse{
			Introduction: intro,
		},
	}, nil
}
*/

func (h *Handler) ReceiveSlackMessage(ctx context.Context, req *connect.Request[api.ReceiveSlackMessageRequest]) (*connect.Response[api.ReceiveSlackMessageResponse], error) {
	log.Debug().Msg("Received Slack message")

	return &connect.Response[api.ReceiveSlackMessageResponse]{
		Msg: &api.ReceiveSlackMessageResponse{
			Challenge: req.Msg.Challenge,
		},
	}, nil
}
