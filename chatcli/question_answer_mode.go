package main

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
)

type QuestionAnswerMode struct {
	chat *Chat
}

func NewQuestionAnswerMode(chat *Chat) *QuestionAnswerMode {
	return &QuestionAnswerMode{
		chat: chat,
	}
}

func (ats *QuestionAnswerMode) Start() (Message, Command, error) {
	return TextMessage("Ask away!"), MODE_START, nil
}

func (m *QuestionAnswerMode) HandleIntent(msg Message) (Message, Command, error) {
	// Refactored to use Message type for input and output
	return m.HandleResponse(msg)
}

func (m *QuestionAnswerMode) HandleResponse(msg Message) (Message, Command, error) {
	// Refactored to use Message type for input and output
	userMessage := msg.Text
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")
	ctx := context.Background()
	res, err := client.AnswerQuestion(ctx, &connect.Request[api.AnswerQuestionRequest]{
		Msg: &api.AnswerQuestionRequest{
			Questions: userMessage,
			UseAi:     true,
		},
	})
	if err != nil {
		return Message{}, NOOP, err
	}

	var responseText string
	for _, v := range res.Msg.Answers {
		responseText += v.Answer + "\n"
	}
	return TextMessage(responseText), NOOP, nil
}

func (ats *QuestionAnswerMode) Stop() error {
	return nil
}
