package chatcli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
	. "github.com/wricardo/code-surgeon/api"

	"github.com/sashabaranov/go-openai"
)

var CODE TMode = "code"

func init() {
	RegisterMode(CODE, NewCodeMode)
}

type CodeMode struct {
	chat *ChatImpl
	form *Form

	ask      StringPromise
	filename StringPromise
}

func NewCodeMode(chat *ChatImpl) *CodeMode {
	codeForm := NewForm()

	return &CodeMode{
		chat:     chat,
		form:     codeForm,
		ask:      codeForm.AddQuestion("What kind of code you want me to write?", true, fnValidateNotEmpty, ""),
		filename: codeForm.AddQuestion("What's the filename for the existing/new code?", true, fnValidateNotEmpty, ""),
	}
}

func (cs *CodeMode) Start() (*Message, *Command, error) {
	return &Message{Form: cs.form.MakeFormMessage()}, MODE_START, nil
}

func (cs *CodeMode) HandleResponse(msg *Message) (*Message, *Command, error) {
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
			return &Message{Form: cs.form.MakeFormMessage()}, NOOP, nil
		}
	}

	// Generate code after form is filled
	response, err := cs.GenerateCode()
	if err != nil {
		return &Message{}, NOOP, err
	}
	return TextMessage(response), MODE_QUIT, nil
}

func (cs *CodeMode) BestShot(msg *Message) (*Message, *Command, error) {
	message, _, err := cs.HandleResponse(msg)
	return message, NOOP, err
}

func (cs *CodeMode) HandleIntent(msg *Message, intent Intent) (*Message, *Command, error) {
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
		return "aider error: " + err.Error(), nil
	}

	return "Instructions: " + aiOut.Instructions + "\n\nAider output:\n" + aiderOut, nil
}

func (cs *CodeMode) Name() string {
	return "code"
}

func (cs *CodeMode) Stop() error {
	return nil
}

// Aider function to execute a CLI command
func Aider(file string, message string) (string, error) {
	// Construct the command with the necessary arguments
	cmd := exec.Command("aider", "--read", "CONVENTIONS.md", "--no-auto-commits", "--gitignore", "--show-diff", "--message", message, "--file", file, "--auto-test", "--test-cmd", "echo 'No tests, ok'", "--architect")

	// Connect the command's stdin to the user's stdin
	cmd.Stdin = os.Stdin

	// Create a buffer to store the output
	var outputBuffer bytes.Buffer

	// Use MultiWriter to write to both the buffer and stdout
	multiWriter := io.MultiWriter(&outputBuffer, os.Stdout)

	// Set the command's stdout and stderr to the multiWriter
	cmd.Stdout = multiWriter
	cmd.Stderr = multiWriter

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to execute aider command: %w\nOutput: %s", err, outputBuffer.String())
	}

	return outputBuffer.String(), nil
}
