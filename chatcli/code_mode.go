package main

import (
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"

	"github.com/sashabaranov/go-openai"
)

type CodeMode struct {
	chat *Chat
	form *PoopForm

	ask      StringPromise
	filename StringPromise
}

func NewCodeMode(chat *Chat) *CodeMode {
	codeForm := NewPoopForm()

	return &CodeMode{
		chat:     chat,
		form:     codeForm,
		ask:      codeForm.AddQuestion("What kind of code you want me to write?", true, fnValidateNotEmpty, ""),
		filename: codeForm.AddQuestion("What's the filename for the existing/new code?", true, fnValidateNotEmpty, ""),
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
	prompt := "Please write a concise instruction for a developer to implement whatever the user is asking. You are not going to write the code, just write the requirement and/or instruction. Output the instruction in a json format, with only one key."

	// Construct the chat messages for the Chat method
	chatMessages := []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: prompt,
		},
		{
			Role:    "user",
			Content: cs.ask(),
		},
	}

	type AiOutput struct {
		Instructions string `json:"instructions" jsonschema:"title=instructions,description=the instructions."`
	}

	// Call the Chat method with the constructed messages
	var aiOut AiOutput
	err := cs.chat.Chat(&aiOut, chatMessages)
	if err != nil {
		return "", err
	}

	if aiOut.Instructions == "" {
		return "No instructions was generated. Please try again.", nil
	}

	// Call the aider function to generate the code
	aiderOut, err := Aider(cs.filename(), aiOut.Instructions)
	if err != nil {
		return "", err
	}

	return "Instructions: " + aiOut.Instructions + "\n\nAider output:\n" + aiderOut, nil
}

func (cs *CodeMode) Stop() error {
	return nil
}
func init() {
	RegisterMode(CODE, NewCodeMode)
}

// Aider function to execute a CLI command
func Aider(file string, message string) (string, error) {
	// Construct the command with the necessary arguments
	cmd := exec.Command("aider", "--yes", "--read", "CONVENTIONS.md", "--auto-commits", "false", "--gitignore", "--show-diff", "--message", message, "--file", file, "--auto-test", "true", "--test-cmd", "echo 'No tests, ok'")

	// Run the command and capture the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute aider command: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}
