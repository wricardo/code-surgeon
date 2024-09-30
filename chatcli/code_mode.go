package main

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/sashabaranov/go-openai"
)

type CodeMode struct {
	chat *Chat
	form *PoopForm
}

func NewCodeMode(chat *Chat) *CodeMode {
	codeForm := NewPoopForm()

	codeForm.AddQuestion("What kind of code you want me to write?", true, fnValidateNotEmpty, "")
	codeForm.AddQuestion("What's the file name for the main.go file?", true, fnValidateNotEmpty, "")

	return &CodeMode{
		chat: chat,
		form: codeForm,
	}
}

func (cs *CodeMode) Start() (Message, Command, error) {
	return Message{Form: cs.form.MakeFormMessage()}, MODE_START, nil
}

func (cs *CodeMode) HandleResponse(msg Message) (Message, Command, error) {
	log.Debug().
		Any("msg", msg).
		Msg("handling response on code")
	if msg.Form != nil || !cs.form.IsFilled() {
		if msg.Form != nil {
			for _, qa := range msg.Form.Questions {
				cs.form.Answer(qa.Question, qa.Answer)
			}
		}
		if !cs.form.IsFilled() {
			return Message{Form: cs.form.MakeFormMessage()}, NOOP, nil
		}
	}

	// Generate code after form is filled
	response, err := cs.GenerateCode()
	if err != nil {
		return Message{}, NOOP, err
	}
	return TextMessage(response), MODE_QUIT, nil
}

func (cs *CodeMode) HandleIntent(msg Message) (Message, Command, error) {
	return cs.HandleResponse(msg)
}

func (cs *CodeMode) GenerateCode() (string, error) {
	// Construct the prompt for the engineer
	prompt := "Please engineer the following code based on the provided answers."
	for _, q := range cs.form.Questions {
		prompt += fmt.Sprintf("\n%s: %s", q.Question, q.Answer)
	}

	// Construct the chat messages for the Chat method
	chatMessages := []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: prompt,
		},
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
func init() {
	RegisterMode(CODE, NewCodeMode)
}
