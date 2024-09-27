package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"github.com/davecgh/go-spew/spew"
	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
)

type CypherMode struct {
	chat *Chat
}

func NewCypherMode(chat *Chat) *CypherMode {
	return &CypherMode{
		chat: chat,
	}
}

func (m *CypherMode) Start() (string, Command, error) {
	return "What's the cypher you'd like to run?", MODE_START, nil
}

func (m *CypherMode) HandleResponse(userMessage string) (string, Command, error) {
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")
	ctx := context.Background()
	var myEnv map[string]string
	myEnv, err := godotenv.Read()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	openaiApiKey, ok := myEnv["OPENAI_API_KEY"]
	if !ok {
		return "", NOOP, fmt.Errorf("OPENAI_API_KEY not found in .env")
	}
	oaiClient := openai.NewClient(openaiApiKey)
	instructorClient := instructor.FromOpenAI(
		oaiClient,
		instructor.WithMode(instructor.ModeJSON),
		instructor.WithMaxRetries(3),
	)

	cypher, followUp, err := GenerateCypher(instructorClient, m.chat.GetConversationSummary(), m.chat.GetHistory(), userMessage)
	if err != nil {
		return "", NOOP, err
	}
	if cypher != "" {
		res, err := client.QueryNeo4J(ctx, &connect.Request[api.QueryNeo4JRequest]{
			Msg: &api.QueryNeo4JRequest{
				Cypher: cypher,
			},
		})
		if err != nil {
			return "oops. We got an error from neo4j: " + err.Error(), NOOP, nil
		}
		log.Println(res.Msg.Output)
		return cypher + "\n" + res.Msg.Output, NOOP, nil
	}

	return followUp, NOOP, nil
}

func GenerateCypher(client *instructor.InstructorOpenAI, summary string, history []Message, ask string) (string, string, error) {
	type AiOutput struct {
		Cypher   string `json:"cypher" jsonschema:"title=cypher,description=the cypher query to be executed on the neo4j database."`
		Question string `json:"question" jsonschema:"title=questtion,description=the question to be asked to the user."`
	}

	ctx := context.Background()
	var aiOut AiOutput
	gptMessages := make([]openai.ChatCompletionMessage, 0, len(history)+2)
	// add history, last 10 messages
	from := len(history) - 10
	if from < 0 {
		from = 0
	}
	history = history[from:]
	for _, msg := range history {
		role := openai.ChatMessageRoleUser
		if msg.Sender == SenderAI {
			role = openai.ChatMessageRoleAssistant
		}
		gptMessages = append(gptMessages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
		})
	}
	gptMessages = append(gptMessages, []openai.ChatCompletionMessage{
		{
			Role:    "user",
			Content: "information for context: " + summary,
		},
		{
			Role:    "system",
			Content: "You are a helpful assistant which can generate cypher queries based on the user's request. You must answer in json format with the key being 'cypher'. If you are not asked to create a cypher or you need more information from the user in order to generate a cypher, you can ask a question to the user, use the key question. Output either a cypher or a question, not both. A question should be asked if you need more information to generate a cypher.",
		},

		{
			Role:    "user",
			Content: ask,
		},
	}...)

	log.Printf("gptMessages\n%s", spew.Sdump(gptMessages)) // TODO: wallace debug

	_, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     openai.GPT4o,
		Messages:  gptMessages,
		MaxTokens: 1000,
	}, &aiOut)

	if err != nil {
		return "", "", fmt.Errorf("Failed to generate cypher query: %v", err)
	}
	log.Printf("aiOUt\n%s", spew.Sdump(aiOut)) // TODO: wallace debug

	if aiOut.Cypher == "" {
		return "", aiOut.Question, nil
	}

	return aiOut.Cypher, "", nil
}

func (m *CypherMode) Stop() error {
	return nil
}
