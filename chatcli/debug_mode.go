package chatcli

import (
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/rs/zerolog/log"
	. "github.com/wricardo/code-surgeon/api"
)

type DebugMode struct {
	chat *ChatImpl
}

func NewDebugMode(chat *ChatImpl) *DebugMode {
	return &DebugMode{
		chat: chat,
	}
}

func (m *DebugMode) HandleIntent(msg *Message, intent Intent) (*Message, *Command, error) {
	fmt.Printf("intent.ParsedIntentAttributes\n%s", spew.Sdump(intent.ParsedIntentAttributes)) // TODO: wallace debug

	var action string
	if tmp, ok := intent.ParsedIntentAttributes["action"]; tmp != "" && ok {
		action = tmp
	}
	if tmp, ok := intent.ParsedIntentAttributes["command"]; tmp != "" && ok {
		action = tmp
	}

	if action == "summary" {
		msg.Text = "summary"
	} else if action == "history" {
		msg.Text = "history"
	} else {
		log.Warn().Str("action", action).Interface("attributes", intent.ParsedIntentAttributes).Msg("invalid command on intent")
		return TextMessage("invalid command on intent. Available commands: summary, history"), NOOP, nil
	}

	res, _, err := m.HandleResponse(msg)
	return res, MODE_QUIT, err
}

func (m *DebugMode) BestShot(msg *Message) (*Message, *Command, error) {
	msg.Text = strings.TrimPrefix(msg.Text, "/debug ")
	message, _, err := m.HandleResponse(msg)
	return message, NOOP, err
}

func (m *DebugMode) Start() (*Message, *Command, error) {
	return TextMessage("you can ask for: summary, history"), NOOP, nil
}

func (m *DebugMode) HandleResponse(msg *Message) (*Message, *Command, error) {
	userMessage := msg.Text
	if userMessage == "summary" {
		return TextMessage(m.chat.GetConversationSummary()), NOOP, nil
	}

	if userMessage == "history" {
		return TextMessage(m.chat.SprintHistory()), NOOP, nil
	}

	return TextMessage("Available commands: summary, history"), NOOP, nil
}

func (m *DebugMode) Stop() error {
	return nil
}

var DEBUG TMode = "debug"

func init() {
	RegisterMode(DEBUG, NewDebugMode)
}
