package main

type DebugMode struct {
	chat *Chat
}

func NewDebugMode(chat *Chat) *DebugMode {
	return &DebugMode{
		chat: chat,
	}
}

func (m *DebugMode) HandleIntent(msg Message) (Message, Command, error) {
	return m.HandleResponse(msg)
}

func (m *DebugMode) Start() (Message, Command, error) {
	return TextMessage("you can ask for: summary, history"), SILENT, nil
}

func (m *DebugMode) HandleResponse(msg Message) (Message, Command, error) {
	userMessage := msg.Text
	if userMessage == "summary" {
		return TextMessage(m.chat.GetConversationSummary()), SILENT, nil
	}

	if userMessage == "history" {
		return TextMessage(m.chat.SprintHistory()), SILENT, nil
	}

	return TextMessage("invalid command. Available commands: summary, history"), SILENT, nil
}

func (m *DebugMode) Stop() error {
	return nil
}
func init() {
	RegisterMode(DEBUG, NewDebugMode)
}
