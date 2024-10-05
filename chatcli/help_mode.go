package chatcli

import (
	"strings"

	. "github.com/wricardo/code-surgeon/api"
)

type HelpMode struct {
	chat *ChatImpl
}

func NewHelpMode(chat *ChatImpl) *HelpMode {
	return &HelpMode{
		chat: chat,
	}
}

func (m *HelpMode) BestShot(msg *Message) (*Message, *Command, error) {
	message, _, err := m.HandleResponse(msg)
	return message, NOOP, err
}

func (m *HelpMode) HandleIntent(msg *Message, intent Intent) (*Message, *Command, error) {
	return m.HandleResponse(msg)
}

func (m *HelpMode) Start() (*Message, *Command, error) {
	commands := []string{"/help", "/exit"}
	for command := range modeKeywords {
		commands = append(commands, command)
	}
	return TextMessage("You can ask any question or run these commands: " + strings.Join(commands, ", ")), MODE_QUIT, nil
}

func (m *HelpMode) HandleResponse(userMessage *Message) (*Message, *Command, error) {
	commands := []string{"/help", "/exit"}
	for command := range modeKeywords {
		commands = append(commands, command)
	}
	return TextMessage("You can ask any question or run these commands: " + strings.Join(commands, ", ")), MODE_QUIT, nil
}

func (m *HelpMode) Stop() error {
	return nil
}

var HELP TMode = "help"

func init() {
	RegisterMode(HELP, NewHelpMode)
}
