package main

import (
	"fmt"

	codesurgeon "github.com/wricardo/code-surgeon"
)

type ParseMode struct {
	chat *Chat
}

func NewParseMode(chat *Chat) *ParseMode {
	return &ParseMode{chat: chat}
}

func (m *ParseMode) Start() (Message, Command, error) {
	// Display a form to the user to get the directory or file path
	form := NewForm([]QuestionAnswer{
		{Question: "Enter the directory or file path to parse:", Answer: ""},
	})
	return Message{Form: form}, NOOP, nil
}

func (m *ParseMode) HandleIntent(msg Message) (Message, Command, error) {
	return m.HandleResponse(msg)
}

func (m *ParseMode) HandleResponse(input Message) (Message, Command, error) {
	if input.Form == nil || len(input.Form.Questions) == 0 {
		return Message{}, NOOP, fmt.Errorf("no input provided")
	}

	fileOrDirectory := input.Form.Questions[0].Answer
	parsedInfo, err := codesurgeon.ParseDirectory(fileOrDirectory)
	if err != nil {
		return Message{Text: fmt.Sprintf("Error parsing: %v", err)}, NOOP, nil
	}

	// Convert parsedInfo to a string or JSON to display to the user
	parsedInfoStr := fmt.Sprintf("Parsed Info: %+v", parsedInfo)
	return Message{Text: parsedInfoStr}, MODE_QUIT, nil
}

func (m *ParseMode) Stop() error {
	return nil
}

func init() {
	RegisterMode("codeparser", NewParseMode)
}
