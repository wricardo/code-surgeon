package chatcli

import (
	"github.com/sashabaranov/go-openai"
	. "github.com/wricardo/code-surgeon/api"
)

var ARCHITECT TMode = "architect"

func init() {
	RegisterMode(ARCHITECT, NewArchitectMode)
}

type ArchitectMode struct {
	chat *ChatImpl
}

func NewArchitectMode(chat *ChatImpl) *ArchitectMode {

	return &ArchitectMode{
		chat: chat,
	}
}

func (as *ArchitectMode) Start() (*Message, *Command, error) {
	return &Message{Text: "let's geek out"}, MODE_START, nil
}

func (as *ArchitectMode) BestShot(msg *Message) (*Message, *Command, error) {
	message, _, err := as.HandleResponse(msg)
	return message, NOOP, err
}

func (as *ArchitectMode) HandleResponse(msg *Message) (*Message, *Command, error) {

	type AiOutput struct {
		Response string `json:"response" jsonschema:"title=response,description=the assistant's response to the user."`
	}
	var aiOut AiOutput

	err := as.chat.Chat(&aiOut, []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "you are a software architect who is very helpful in discussing software design and architecture. You should try to stick to the topic of software design and architecture.Pay attention to the history and summary.",
		},
		{
			Role:    "user",
			Content: msg.Text,
		},
	})
	if err != nil {
		return TextMessage("chat error: " + err.Error()), MODE_QUIT, nil
	}
	return TextMessage(aiOut.Response), NOOP, nil
}

func (as *ArchitectMode) HandleIntent(msg *Message, intent Intent) (*Message, *Command, error) {
	return as.HandleResponse(msg)
}

func (as *ArchitectMode) Stop() error {
	// do nothing
	return nil
}
