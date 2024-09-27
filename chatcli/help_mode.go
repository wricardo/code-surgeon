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

func (m *HelpMode) Start() (string, Command, error) {
	commands := []string{"help", "exit"}
	for command := range modeKeywords {
		commands = append(commands, command)
	}
	return "You can ask any question or run these commands: " + strings.Join(commands, ", "), MODE_QUIT, nil
}

func (m *HelpMode) HandleResponse(userMessage string) (string, Command, error) {
	return "", MODE_QUIT, nil
}

func (m *HelpMode) Stop() error {
	return nil
}
