package grpc

import (
	"context"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"connectrpc.com/connect"
	"github.com/Jeffail/gabs"
	"github.com/davecgh/go-spew/spew"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
	"github.com/wricardo/code-surgeon/chatcli"
)

var _ apiconnect.GptServiceHandler = (*Handler)(nil)

type Handler struct {
	publicUrl string
	// chat      chatcli.IChat
	mu sync.Mutex // protects the chat
}

func (h *Handler) NewChat(ctx context.Context, req *connect.Request[api.NewChatRequest]) (*connect.Response[api.NewChatResponse], error) {
	// Create a new Chat instance
	newChat := &api.Chat{
		Id:       uuid.New().String(),
		Messages: []*api.Message{},
	}

	// Create the response
	response := &api.NewChatResponse{
		Chat: newChat,
	}

	// Return the response
	return &connect.Response[api.NewChatResponse]{Msg: response}, nil
}

func NewHandler(publicUrl string) *Handler {
	h := &Handler{
		publicUrl: publicUrl,
	}
	return h
}

func (h *Handler) SendMessage(ctx context.Context, req *connect.Request[api.SendMessageRequest]) (*connect.Response[api.SendMessageResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	res, cmd, err := chatcli.GLOBAL_CHAT.HandleUserMessage(req.Msg.Message)
	if err != nil {
		log.Error().Err(err).Msg("Error sending message")
	}
	if cmd != nil && cmd.Name != "" {
		// FIXME: handle commands
		if cmd.Name == "exit" {
			// h.chat.Stop()
		}
	}

	modeName := chatcli.GLOBAL_CHAT.GetModeText()
	if req.Msg.Message.Text == "/debug" {
		modeName = "debug"
	}

	response := &api.SendMessageResponse{
		Command: cmd,
		Message: res,
		Mode: &api.Mode{
			Name: modeName,
		},
	}

	return &connect.Response[api.SendMessageResponse]{Msg: response}, nil
}

func (h *Handler) GetChat(ctx context.Context, req *connect.Request[api.GetChatRequest]) (*connect.Response[api.GetChatResponse], error) {

	// Create an empty Chat instance
	emptyChat := &api.Chat{
		Id:       "",               // Empty ID
		Messages: []*api.Message{}, // Empty list of messages
	}

	// Create the response
	response := &api.GetChatResponse{
		Chat: emptyChat,
	}

	GLOBAL_CHAT := chatcli.GLOBAL_CHAT
	if GLOBAL_CHAT != nil {
		for _, msg := range GLOBAL_CHAT.GetHistory() {
			response.Chat.Messages = append(response.Chat.Messages, msg)
		}
		if GLOBAL_CHAT.GetModeText() != "" {
			response.Chat.CurrentMode = &api.Mode{
				Name: GLOBAL_CHAT.GetModeText(),
			}

		}

	}

	// Return the response
	return &connect.Response[api.GetChatResponse]{Msg: response}, nil
}

/*
func (*Handler) SearchForGolangFunction(ctx context.Context, req *connect.Request[api.SearchForGolangFunctionRequest]) (*connect.Response[api.SearchForGolangFunctionResponse], error) {
	path := req.Msg.Path
	if path == "" {
		path = "."
	}

	path, err := codesurgeon.FindFunction(path, req.Msg.Receiver, req.Msg.FunctionName)
	if err != nil {
		log.Printf("Error searching for function: %v", err)
		return &connect.Response[api.SearchForGolangFunctionResponse]{
			Msg: &api.SearchForGolangFunctionResponse{},
		}, nil
	}
	if path == "" {
		log.Printf("Function not found")
		return &connect.Response[api.SearchForGolangFunctionResponse]{
			Msg: &api.SearchForGolangFunctionResponse{},
		}, nil
	}

	parsedInfo, err := codesurgeon.ParseDirectory(path)
	if err != nil {
		log.Printf("Error parsing directory: %v", err)
		return &connect.Response[api.SearchForGolangFunctionResponse]{
			Msg: &api.SearchForGolangFunctionResponse{},
		}, nil
	}
	if len(parsedInfo.Packages) == 0 {
		log.Printf("No packages found")
		return &connect.Response[api.SearchForGolangFunctionResponse]{
			Msg: &api.SearchForGolangFunctionResponse{},
		}, nil
	}

	msg := &api.SearchForGolangFunctionResponse{
		Filepath: path,
		// Signature:     fn.Signature,
		// Documentation: strings.Join(fn.Docs, "\n"),
		// Body:          fn.Body,
	}

	// fmt.Printf("parsedInfo\n%s\n", spew.Sdump(parsedInfo))

	for _, pkg := range parsedInfo.Packages {
		if req.Msg.Receiver != "" {
			for _, st := range pkg.Structs {
				if st.Name == req.Msg.Receiver {
					for _, f := range st.Methods {
						if f.Name == req.Msg.FunctionName {
							msg.Signature = f.Signature
							msg.Documentation = strings.Join(f.Docs, "\n")
							msg.Body = f.Body
							break
						}
					}
				}
			}
		} else {
			for _, f := range pkg.Functions {
				fmt.Println(f.Name, req.Msg.FunctionName)
				if f.Name == req.Msg.FunctionName {
					msg.Signature = f.Signature
					msg.Documentation = strings.Join(f.Docs, "\n")
					msg.Body = f.Body
					break
				}

			}
		}
	}

	return &connect.Response[api.SearchForGolangFunctionResponse]{
		Msg: msg,
	}, nil
}

func (_ *Handler) UpsertDocumentationToFunction(ctx context.Context, req *connect.Request[api.UpsertDocumentationToFunctionRequest]) (*connect.Response[api.UpsertDocumentationToFunctionResponse], error) {
	msg := req.Msg
	ok, err := codesurgeon.UpsertDocumentationToFunction(msg.Filepath, msg.Receiver, msg.FunctionName, msg.Documentation)
	if err != nil {
		return nil, err
	}

	return &connect.Response[api.UpsertDocumentationToFunctionResponse]{
		Msg: &api.UpsertDocumentationToFunctionResponse{
			Ok: ok,
		},
	}, nil
}

func (*Handler) UpsertCodeBlock(ctx context.Context, req *connect.Request[api.UpsertCodeBlockRequest]) (*connect.Response[api.UpsertCodeBlockResponse], error) {
	msg := req.Msg
	changes := []codesurgeon.FileChange{}

	block := msg.Modification
	// for _, block := range msg.Modification {
	change := codesurgeon.FileChange{
		PackageName: block.PackageName,
		File:        block.Filepath,
		Fragments: []codesurgeon.CodeFragment{
			{
				Content:   block.CodeBlock,
				Overwrite: block.Overwrite,
			},
		},
	}
	changes = append(changes, change)
	// }
	err := codesurgeon.ApplyFileChanges(changes)
	if err != nil {
		log.Printf("Error applying file changes: %v\n", err)
		return &connect.Response[api.UpsertCodeBlockResponse]{
			Msg: &api.UpsertCodeBlockResponse{
				Ok: false,
			},
		}, nil
	}

	return &connect.Response[api.UpsertCodeBlockResponse]{
		Msg: &api.UpsertCodeBlockResponse{
			Ok: true,
		},
	}, nil
}

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

func (h *Handler) GetOpenAPI(ctx context.Context, req *connect.Request[api.GetOpenAPIRequest]) (*connect.Response[api.GetOpenAPIResponse], error) {
	// Read the embedded file using the embedded FS
	data, err := codesurgeon.FS.ReadFile("api/codesurgeon.openapi.json")
	if err != nil {
		return nil, err
	}

	parsed, err := gabs.ParseJSON(data)
	if err != nil {
		return nil, err
	}
	// https://chatgpt.com/gpts/editor/g-v09HRlzOu

	// add "server" field
	url := h.publicUrl
	url = strings.TrimSuffix("https://"+url, "/")
	log.Printf("url: %s", spew.Sdump(h.publicUrl))

	parsed.Array("servers")
	parsed.ArrayAppend(map[string]string{
		"url": url,
	}, "servers")

	//
	// Update "openapi" field to "3.1.0"
	parsed.Set("3.1.0", "openapi")

	// Paths to check
	paths, err := parsed.Path("paths").ChildrenMap()
	if err != nil {
		return nil, err
	}

	// Iterate over paths to update "operationId"
	for _, path := range paths {
		// Get the "post" object within each path
		post := path.Search("post")
		if post != nil {

			post.Set("false", "x-openai-isConsequential")

			// Get current "operationId"
			operationID, ok := post.Path("operationId").Data().(string)
			if ok {
				// Split the "operationId" by "."
				parts := strings.Split(operationID, ".")
				operationID := "operationId"
				// Get the last 2 parts of the "operationId" and join them with a "_"
				if len(parts) > 1 {
					operationID = strings.Join(parts[len(parts)-2:], "_")
				} else if len(parts) > 0 {
					operationID = parts[0]
				}
				operationID = strings.TrimPrefix(operationID, "GptService_")

				// Update "operationId"
				post.Set(operationID, "operationId")
			}
		}
	}

	return &connect.Response[api.GetOpenAPIResponse]{
		Msg: &api.GetOpenAPIResponse{
			Openapi: parsed.String(),
		},
	}, nil
}
