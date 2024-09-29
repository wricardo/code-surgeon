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

func (m *CypherMode) Start() (Message, Command, error) {
	return TextMessage("What's the cypher you'd like to run?"), MODE_START, nil
}

func (m *CypherMode) HandleIntent(msg Message) (Message, Command, error) {
	return m.HandleResponse(msg)
}

func (m *CypherMode) HandleResponse(msg Message) (Message, Command, error) {
	userMessage := msg.Text
	type AiOutput struct {
		Cypher   string `json:"cypher" jsonschema:"title=cypher,description=the cypher query to be executed on the neo4j database."`
		Question string `json:"question" jsonschema:"title=questtion,description=the question to be asked to the user."`
	}
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")
	ctx := context.Background()

	var aiOut AiOutput
	err := m.chat.Chat(&aiOut, []openai.ChatCompletionMessage{
		{
			Role: "system",
			Content: `You are a helpful assistant which can generate cypher queries based on the user's request. 
			You must answer in json.
			If the user is asking for a cypher query, you must provide the cypher query.
			Ask clarifying questions if needed, let's talk through what the user wants to query.
			`,
		},

		{
			Role:    "user",
			Content: userMessage,
		},
	})
	if err != nil {
		return Message{}, NOOP, err
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
			return TextMessage("oops. We got an error from neo4j: " + err.Error()), NOOP, nil
		}
		log.Println(res.Msg.Output)
		return TextMessage(cypher + "\n" + res.Msg.Output), NOOP, nil
	}

	return TextMessage(followUp), NOOP, nil
}

func (m *CypherMode) Stop() error {
	return nil
}
