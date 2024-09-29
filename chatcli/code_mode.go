package main

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/sashabaranov/go-openai"
)

type CodeMode struct {
	questionAnswerMap map[string]string
	questions         []string
	questionIndex     int
	chat              *Chat
}

func NewCodeMode(chat *Chat) *CodeMode {
	return &CodeMode{
		questionAnswerMap: make(map[string]string),
		questions: []string{
			"What kind of code you want me to write?",
			"What's the file name for the main.go file?",
		},
		questionIndex: 0,
		chat:          chat,
	}
}

func (cs *CodeMode) Start() (Message, Command, error) {
	question, _, _ := cs.AskNextQuestion()
	return question, MODE_START, nil
}

func (cs *CodeMode) HandleResponse(msg Message) (Message, Command, error) {
	log.Debug().
		Any("msg", msg).
		Any("questionIndex", cs.questionIndex).
		Any("questions", cs.questions).
		Int("lenQuestions", len(cs.questions)).
		Any("questionAnswerMap", cs.questionAnswerMap).
		Msg("handling response on code")
	if msg.Text == "" && msg.Form == nil {
		log.Print("empty message on code, starting next question")
		question, command, _ := cs.AskNextQuestion()
		return question, command, nil
	}

	if msg.Form != nil {
		log.Print("handling form")
		return cs.HandleForm(*msg.Form)

	}

	if cs.questionIndex <= len(cs.questions) {
		log.Print("asking question")
		question, command, _ := cs.AskNextQuestion()
		return question, command, nil
	}

	log.Print("generating code HEREEEE")

	type AiOutput struct {
		Response string `json:"response" jsonschema:"title=response,description=the response from the AI model."`
	}

	var aiOut AiOutput
	err := cs.chat.Chat(&aiOut, []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are a helpful assistant which can generate code based on the user's request. you must answer in json. Ask clarifying questions if nedded, let's talk through what the user want's to code.",
		},
		{
			Role:    "user",
			Content: msg.Text,
		},
	})

	if err != nil {
		return Message{}, NOOP, err
	}
	return TextMessage(aiOut.Response), NOOP, nil
}

func (cs *CodeMode) HandleForm(form Form) (Message, Command, error) {
	for _, qa := range form.Questions {
		cs.questionAnswerMap[qa.Question] = qa.Answer
	}
	cs.questionIndex++
	if cs.questionIndex < len(cs.questions) {
		question, command, _ := cs.AskNextQuestion()
		return question, command, nil
	} else {
		response, _ := cs.GenerateCode()
		return TextMessage(response), MODE_QUIT, nil
	}
}

func (cs *CodeMode) HandleIntent(msg Message) (Message, Command, error) {
	return cs.HandleResponse(msg)
}

func (cs *CodeMode) AskNextQuestion() (Message, Command, error) {
	if cs.questionIndex >= len(cs.questions) {
		response, _ := cs.GenerateCode()
		return TextMessage(response), MODE_QUIT, nil
	}
	question := cs.questions[cs.questionIndex]

	form := NewForm([]QuestionAnswer{
		{
			Question: question,
			Answer:   "",
		},
	})
	return Message{Form: form}, NOOP, nil
}

func (cs *CodeMode) GenerateCode() (string, error) {
	// Construct the prompt for the engineer
	prompt := "Please engineer the following code based on the provided answers."
	for _, q := range cs.questions {
		prompt += fmt.Sprintf("\n%s: %s", q, cs.questionAnswerMap[q])
	}

	// Construct the chat messages for the Chat method
	chatMessages := []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: prompt,
		},
		// Add more messages if needed
	}

	type AiOutput struct {
		Message string `json:"message" jsonschema:"title=message,description=the message from the AI model."`
	}

	// Call the Chat method with the constructed messages
	var aiOut AiOutput
	err := cs.chat.Chat(&aiOut, chatMessages)
	if err != nil {
		return "", err
	}

	if aiOut.Message == "" {
		return "No code was generated. Please try again.", nil
	}

	return aiOut.Message, nil
}

func (cs *CodeMode) Stop() error {
	return nil
}
