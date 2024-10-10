package chatcli

import (
	"fmt"
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
	return m.returnHelp(), MODE_QUIT, nil
}

func (m *HelpMode) HandleResponse(userMessage *Message) (*Message, *Command, error) {
	return m.returnHelp(), MODE_QUIT, nil
}

func (m *HelpMode) returnHelp() *Message {
	commands := []string{"/help", "/exit"}
	for command := range modeKeywords {
		commands = append(commands, command)
	}

	chatIdMsg := fmt.Sprintf("ChatId: %s\n", m.chat.Id)

	if m.chat != nil {
		if m.chat.modeManager.currentMode != nil {
			commands = append(commands, "/quit")
			return TextMessage(fmt.Sprintf("%sYou are in %s mode.\nAvailable commands: %s", chatIdMsg, m.chat.modeManager.currentMode.Name(), strings.Join(commands, ", ")))
		}
	}
	return TextMessage(chatIdMsg + "You can ask any question or run these commands: " + strings.Join(commands, ", "))
}

func (m *HelpMode) Name() string {
	return "help"
}

func (m *HelpMode) Stop() error {
	return nil
}

var HELP TMode = "help"

func init() {
	RegisterMode(HELP, NewHelpMode)
}
