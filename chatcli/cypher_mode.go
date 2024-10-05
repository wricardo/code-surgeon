package chatcli

import (
	"context"
	"encoding/json"

	"github.com/sashabaranov/go-openai"
	. "github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/neo4j2"
)

type CypherMode struct {
	chat *ChatImpl
}

func NewCypherMode(chat *ChatImpl) *CypherMode {
	return &CypherMode{
		chat: chat,
	}
}

func (m *CypherMode) Start() (*Message, *Command, error) {
	result, err := neo4j2.QueryNeo4J(context.Background(), *m.chat.driver, "CALL db.labels() YIELD label RETURN label", nil)
	if err != nil {
		return TextMessage("oops. We got an error from neo4j: " + err.Error()), NOOP, nil
	}
	// Convert the result to a string or appropriate format
	resultString, err := json.Marshal(result)
	if err != nil {
		return TextMessage("Failed to encode result: " + err.Error()), NOOP, nil
	}

	responseMessage := "Available labels in the database are: \n" + string(resultString)
	responseMessage += "\n\nWhat's the cypher you'd like to run?"

	return TextMessage(responseMessage), MODE_START, nil
}

func (m *CypherMode) BestShot(msg *Message) (*Message, *Command, error) {
	message, _, err := m.HandleResponse(msg)
	return message, NOOP, err
}

func (m *CypherMode) HandleIntent(msg *Message, intent Intent) (*Message, *Command, error) {
	message, command, err := m.HandleResponse(msg)
	return message, command, err
}

func (m *CypherMode) HandleResponse(msg *Message) (*Message, *Command, error) {
	userMessage := msg.Text
	type AiOutput struct {
		Cypher   string `json:"cypher" jsonschema:"title=cypher,description=the cypher query to be executed on the neo4j database."`
		Question string `json:"question" jsonschema:"title=questtion,description=the question to be asked to the user."`
	}
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
		return nil, NOOP, err
	}

	followUp := aiOut.Question
	cypher := aiOut.Cypher

	if cypher != "" {
		// Directly query Neo4j using the chat's Neo4j driver
		result, err := neo4j2.QueryNeo4J(context.Background(), *m.chat.driver, cypher, nil)
		if err != nil {
			return TextMessage("oops. We got an error from neo4j: " + err.Error()), NOOP, nil
		}

		// Convert the result to a string or appropriate format
		resultString, err := json.Marshal(result)
		if err != nil {
			return TextMessage("Failed to encode result: " + err.Error()), NOOP, nil
		}

		return TextMessage(cypher + "\n" + string(resultString)), NOOP, nil
	}

	return TextMessage(followUp), NOOP, nil
}

func (m *CypherMode) Stop() error {
	return nil
}
func init() {
	RegisterMode("cypher", NewCypherMode)
}
