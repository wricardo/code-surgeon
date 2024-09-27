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

func (ats *QuestionAnswerMode) Start() (string, Command, error) {
	return "Ask away!", MODE_START, nil
}

func (ats *QuestionAnswerMode) HandleResponse(userMessage string) (string, Command, error) {
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")
	ctx := context.Background()
	res, err := client.AnswerQuestion(ctx, &connect.Request[api.AnswerQuestionRequest]{
		Msg: &api.AnswerQuestionRequest{
			Questions: userMessage,
			UseAi:     true,
		},
	})
	if err != nil {
		return "", NOOP, err
	}

	var response string
	for _, v := range res.Msg.Answers {
		response += v.Answer + "\n"
	}
	return response, NOOP, nil
}

func (ats *QuestionAnswerMode) Stop() error {
	return nil
}
