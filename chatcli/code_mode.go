package main

import (
	"fmt"
	"strings"
)

type CodeMode struct {
	questionAnswerMap map[string]string
	questions         []string
	questionIndex     int
}

func NewCodeMode(chat *Chat) *CodeMode {
	return &CodeMode{
		questionAnswerMap: make(map[string]string),
		questions: []string{
			"What kind of api would you like to build?",
			"What's the file name for the main.go file?",
		},
		questionIndex: 0,
	}
}

func (cs *CodeMode) Start() (string, Command, error) {
	question, _, _ := cs.AskNextQuestion()
	return "Starting code mode. I will ask you some questions to generate code.\n" + question, MODE_START, nil
}

func (cs *CodeMode) HandleResponse(userMessage string) (string, Command, error) {
	trimmedInput := strings.TrimSpace(userMessage)
	if trimmedInput == "" {
		question, command, _ := cs.AskNextQuestion()
		return "Input cannot be empty. Please provide a valid response.\n" + question, command, nil
	}

	cs.questionAnswerMap[cs.questions[cs.questionIndex]] = userMessage
	cs.questionIndex++
	if cs.questionIndex < len(cs.questions) {
		question, command, _ := cs.AskNextQuestion()
		return question, command, nil
	} else {
		response, _ := cs.GenerateCode()
		return response, MODE_QUIT, nil
	}
}

func (cs *CodeMode) AskNextQuestion() (string, Command, error) {
	if cs.questionIndex >= len(cs.questions) {
		response, _ := cs.GenerateCode()
		return response, MODE_QUIT, nil
	}
	question := cs.questions[cs.questionIndex]
	return question, NOOP, nil
}

func (cs *CodeMode) GenerateCode() (string, error) {
	codeSnippet := ""
	codeSnippet += fmt.Sprintf("// generated based on these questions:\n")
	for _, q := range cs.questions {
		codeSnippet += fmt.Sprintf("// %s: %s\n", q, cs.questionAnswerMap[q])
	}
	codeSnippet += "<<GENERATED CODE>>"

	return "Thank you for the information. Generating code...\n\n" + codeSnippet, nil
}

func (cs *CodeMode) Stop() error {
	return nil
}
