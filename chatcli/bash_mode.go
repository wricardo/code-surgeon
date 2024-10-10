package chatcli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
	. "github.com/wricardo/code-surgeon/api"
)

type BashMode struct {
	chat *ChatImpl
}

func NewBashMode(chat *ChatImpl) *BashMode {
	return &BashMode{chat: chat}
}

func (bm *BashMode) Start() (*Message, *Command, error) {
	return TextMessage("Enter a bash command or describe what you want to do:"), MODE_START, nil
}

func (bm *BashMode) HandleResponse(msg *Message) (*Message, *Command, error) {
	command := strings.TrimSpace(msg.Text)
	if command == "" {
		return TextMessage("Please enter a valid bash command or description."), NOOP, nil
	}

	// If direct execution fails, use LLM to generate a command
	generatedCommand, err := bm.generateCommandWithLLM(command)
	if err != nil {
		return TextMessage(fmt.Sprintf("Error generating command: %v", err)), NOOP, nil
	}

	// Execute the generated command
	output, err := bm.executeCommand(generatedCommand)
	if err != nil {
		return TextMessage(fmt.Sprintf("Error executing generated command: %v", err)), NOOP, nil
	}

	if generatedCommand == command {
		return TextMessage(fmt.Sprintf("Command output:\n%s", output)), NOOP, nil
	}
	return TextMessage(fmt.Sprintf("Generated command:\n%s\n\nOutput:\n%s", generatedCommand, output)), NOOP, nil
}

func (bm *BashMode) BestShot(msg *Message) (*Message, *Command, error) {
	message, _, err := bm.HandleResponse(msg)
	return message, NOOP, err
}

func (bm *BashMode) HandleIntent(msg *Message, intent Intent) (*Message, *Command, error) {
	return bm.HandleResponse(msg)
}

func (bm *BashMode) Name() string {
	return "bash"
}

func (bm *BashMode) Stop() error {
	return nil
}

func (bm *BashMode) executeCommand(command string) (string, error) {
	log.Debug().Str("command", command).Msg("executing command")
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (bm *BashMode) generateCommandWithLLM(userInput string) (string, error) {
	// Use the chat's AI client to generate the command
	type AiOutput struct {
		Command string `json:"command" jsonschema:"title=command,description=the generated bash command."`
	}
	// If asked to grep for a file suggest: fzf --filter '<input>'
	var aiOut AiOutput
	err := bm.chat.Chat(&aiOut, []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are a helpful assistant and an expert in bash commands. You are asked to identify bash commands the user gives and return them or if given a description, generate a bash command for the task. Respond in json. If asked to fine files, ignore the .git directory or other known common directory that has many files that might be in this project.",
		},
		{
			Role:    "user",
			Content: userInput,
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate command: %w", err)
	}

	return aiOut.Command, nil
}

func init() {
	RegisterMode("bash", NewBashMode)
}
