package main

import (
	"context"
	"log"
	"net/http"

	"connectrpc.com/connect"
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
	type AiOutput struct {
		Cypher   string `json:"cypher" jsonschema:"title=cypher,description=the cypher query to be executed on the neo4j database."`
		Question string `json:"question" jsonschema:"title=questtion,description=the question to be asked to the user."`
	}
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")
	ctx := context.Background()

	var aiOut AiOutput
	err := m.chat.Chat(&aiOut, []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are a helpful assistant which can generate cypher queries based on the user's request. You must answer in json format with the key being 'cypher'. If you are not asked to create a cypher or you need more information from the user in order to generate a cypher, you can ask a question to the user, use the key question. Output either a cypher or a question, not both. A question should be asked if you need more information to generate a cypher.",
		},

		{
			Role:    "user",
			Content: userMessage,
		},
	})
	if err != nil {
		return "", NOOP, err
	}

	followUp := aiOut.Question
	cypher := aiOut.Cypher

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

func (m *CypherMode) Stop() error {
	return nil
}
