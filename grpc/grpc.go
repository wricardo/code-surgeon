package grpc

import (
	"context"
	"fmt"
	"log"
	"strings"

	"connectrpc.com/connect"
	"github.com/Jeffail/gabs"
	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sashabaranov/go-openai"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
	"github.com/wricardo/code-surgeon/neo4j2"
)

var _ apiconnect.GptServiceHandler = (*Handler)(nil)

type Handler struct {
	url              string
	neo4jDriver      neo4j.DriverWithContext
	instructorClient *instructor.InstructorOpenAI
	openaiClient     *openai.Client
}

func NewHandler(url string, ic *instructor.InstructorOpenAI, oc *openai.Client, driver neo4j.DriverWithContext) *Handler {

	return &Handler{
		url:              url,
		neo4jDriver:      driver,
		instructorClient: ic,
		openaiClient:     oc,
	}
}

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
	url := h.url
	url = strings.TrimSuffix(url, "/")

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

func (h *Handler) AnswerQuestion(ctx context.Context, req *connect.Request[api.AnswerQuestionRequest]) (*connect.Response[api.AnswerQuestionResponse], error) {

	res := &api.AnswerQuestionResponse{
		Answers: []*api.AnswerQuestionResponse_Answer{},
	}
	userEmbedding, err := ai.EmbedQuestion(h.openaiClient, req.Msg.Questions)
	if err != nil {
		return nil, err
	}
	if len(userEmbedding) == 0 {
		return nil, fmt.Errorf("Failed to embed question, zero length vector returned")
	}

	topQuestionIds, err := neo4j2.VectorSearchQuestions(ctx, h.neo4jDriver, userEmbedding)
	if err != nil {
		return nil,
			err
	}

	topAnswers, err := neo4j2.GetTopAnswersForQuestions(ctx, h.neo4jDriver,
		topQuestionIds,
	)
	if err != nil {
		return nil, err
	}
	finalAnswer, err := neo4j2.GenerateFinalAnswer(h.instructorClient, req.Msg.Questions, topAnswers)
	if err !=
		nil {
		return nil, err
	}
	res.Answers = append(res.Answers, &api.AnswerQuestionResponse_Answer{
		Answer:   finalAnswer,
		Question: req.Msg.Questions,
	})

	return &connect.Response[api.AnswerQuestionResponse]{
		Msg: res,
	}, nil
}
func (h *Handler) SaveToKnowledgeBase(ctx context.
	Context, req *connect.
	Request[api.SaveToKnowledgeBaseRequest]) (*connect.Response[api.SaveToKnowledgeBaseResponse], error) {
	conversationSummary := req.Msg.ConversationSummary
	dateISO := req.Msg.DateIso
	err := neo4j2.SaveConversationSummary(ctx, h.neo4jDriver, conversationSummary,

		dateISO)
	if err != nil {
		log.Printf("Error saving conversation summary to database: %v",

			err)
		return &connect.Response[api.
			SaveToKnowledgeBaseResponse]{Msg: &api.SaveToKnowledgeBaseResponse{
			Ok: false}}, err
	}
	questionsAndAnswers, err := h.
		generateQuestionsAndAnswers(ctx,
			conversationSummary,
		)
	if err != nil {
		log.
			Printf("Error generating questions and answers from conversation summary: %v",

				err)
		return &connect.Response[api.SaveToKnowledgeBaseResponse]{Msg: &api.SaveToKnowledgeBaseResponse{Ok: false}}, err
	}

	for _, qa := range questionsAndAnswers {
		embedding, err := ai.EmbedQuestion(h.openaiClient, qa.Question)
		if err != nil {
			log.Printf("Error embedding question: %v", err)
		}
		err = neo4j2.CreateQuestionAndAnswers(ctx,
			h.neo4jDriver,
			qa.Question,
			embedding,
			qa.Answers)
		if err != nil {
			log.Printf("Error saving generated question and answers to Neo4j: %v",

				err)
			return &connect.Response[api.SaveToKnowledgeBaseResponse]{Msg: &api.SaveToKnowledgeBaseResponse{Ok: false}}, err
		}
	}
	return &connect.Response[api.SaveToKnowledgeBaseResponse]{Msg: &api.SaveToKnowledgeBaseResponse{Ok: true}}, nil
}

func (h *Handler) generateQuestionsAndAnswers(ctx context.Context, conversationSummary string) ([]QuestionAndAnswers, error) {
	// Define the structure for AI output
	type AiOutput struct {
		QuestionsAndAnswers []QuestionAndAnswers `json:"questions_and_answers"`
	}

	// Prepare the prompt
	prompt := "Generate questions and answers so that my bot build a solid knowledge based on the following conversation summary:\n\n" + conversationSummary

	// Initialize the AI output
	var aiOut AiOutput

	// Call OpenAI API to generate questions and answers
	_, err := h.instructorClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4o, // Adjust the model as per requirements
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
		&aiOut,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate questions and answers: %v", err)
	}

	return aiOut.QuestionsAndAnswers, nil
}

// Helper function to parse the OpenAI response into questions and answers (if needed)
func parseQuestionsAndAnswers(response string) []QuestionAndAnswers {
	questionsAndAnswers := []QuestionAndAnswers{}
	// Implement parsing logic here if needed
	return questionsAndAnswers
}

// QuestionAndAnswers struct to hold generated questions and answers
type QuestionAndAnswers struct {
	Question string   `json:"question"`
	Answers  []string `json:"answers"`
}