package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type ProtoMode struct {
	chat              *Chat
	questionAnswerMap map[string]string
	questions         []string
	questionIndex     int
	protoContent      string
}

func NewProtoMode(chat *Chat) *ProtoMode {
	return &ProtoMode{
		chat:              chat,
		questionAnswerMap: make(map[string]string),
		questions: []string{
			"Please provide the path to the PostgreSQL proto file:",
		},
		questionIndex: 0,
	}
}

func (pm *ProtoMode) Start() (Message, Command, error) {
	question, _, _ := pm.AskNextQuestion()
	return question, MODE_START, nil
}

func (pm *ProtoMode) HandleIntent(msg Message) (Message, Command, error) {
	return pm.HandleResponse(msg)
}

func (pm *ProtoMode) HandleResponse(msg Message) (Message, Command, error) {
	if msg.Form != nil {
		return pm.HandleForm(*msg.Form)
	}

	if pm.questionIndex < len(pm.questions) {
		question, command, _ := pm.AskNextQuestion()
		return question, command, nil
	}

	if pm.questionIndex == len(pm.questions) {
		filePath := pm.questionAnswerMap["Please provide the path to the PostgreSQL proto file:"]
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return TextMessage("Failed to read proto file: " + err.Error()), NOOP, nil
		}
		pm.protoContent = string(content)
		apiKeys := pm.extractApiKeys(pm.protoContent)
		pm.questions = append(pm.questions, apiKeys...)
		pm.questionIndex++
	}

	if pm.questionIndex > len(pm.questions) {
		return pm.GenerateApiCall()
	}

	return Message{}, NOOP, nil
}

func (pm *ProtoMode) HandleForm(form FormMessage) (Message, Command, error) {
	for _, qa := range form.Questions {
		pm.questionAnswerMap[qa.Question] = qa.Answer
	}
	pm.questionIndex++
	if pm.questionIndex < len(pm.questions) {
		question, command, _ := pm.AskNextQuestion()
		return question, command, nil
	}
	return pm.HandleResponse(Message{})
}

func (pm *ProtoMode) AskNextQuestion() (Message, Command, error) {
	if pm.questionIndex >= len(pm.questions) {
		return Message{}, MODE_QUIT, nil
	}
	question := pm.questions[pm.questionIndex]

	form := NewForm([]QuestionAnswer{
		{
			Question: question,
			Answer:   "",
		},
	})
	return Message{Form: form}, NOOP, nil
}

func (pm *ProtoMode) extractApiKeys(protoContent string) []string {
	lines := strings.Split(protoContent, "\n")
	apiKeys := []string{}
	for _, line := range lines {
		if strings.Contains(line, "rpc") {
			apiKeys = append(apiKeys, "Enter value for "+strings.TrimSpace(line))
		}
	}
	return apiKeys
}

func (pm *ProtoMode) GenerateApiCall() (Message, Command, error) {
	type AiOutput struct {
		Response string `json:"response" jsonschema:"title=response,description=the response from the AI model."`
	}

	var aiOut AiOutput
	err := pm.chat.Chat(&aiOut, []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are a helpful assistant which can generate API calls based on the user's request and proto file content. You must answer in json.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Generate an API call using the following inputs: %v", pm.questionAnswerMap),
		},
	})

	if err != nil {
		return Message{}, NOOP, err
	}
	return TextMessage(aiOut.Response), NOOP, nil
}

func (pm *ProtoMode) Stop() error {
	return nil
}

func init() {
	RegisterMode("proto", NewProtoMode)
}
