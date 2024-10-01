package main

import "strings"

type HelpMode struct {
	chat *Chat
}

func NewHelpMode(chat *Chat) *HelpMode {
	return &HelpMode{
		chat: chat,
	}
}

func (m *HelpMode) HandleIntent(msg Message) (Message, Command, error) {
	return m.HandleResponse(msg)
}

func (m *HelpMode) Start() (Message, Command, error) {
	commands := []string{"/help", "/exit"}
	for command := range modeKeywords {
		commands = append(commands, command)
	}
	return TextMessage("You can ask any question or run these commands: " + strings.Join(commands, ", ")), MODE_QUIT, nil
}

func (m *HelpMode) HandleResponse(userMessage Message) (Message, Command, error) {
	return Message{}, MODE_QUIT, nil
}

func (m *HelpMode) Stop() error {
	return nil
}
func init() {
	RegisterMode(HELP, NewHelpMode)
}
