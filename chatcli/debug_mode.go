package main

type DebugMode struct {
	chat *Chat
}

func NewDebugMode(chat *Chat) *DebugMode {
	return &DebugMode{
		chat: chat,
	}
}

func (m *DebugMode) Start() (string, Command, error) {
	return "you can ask for: summary, history", MODE_START, nil
}

func (m *DebugMode) HandleResponse(userMessage string) (string, Command, error) {
	if userMessage == "summary" {
		return m.chat.GetConversationSummary(), NOOP, nil
	}

	if userMessage == "history" {
		return m.chat.SprintHistory(), NOOP, nil
	}
	return "exiting debug mode", MODE_QUIT, nil
}

func (m *DebugMode) Stop() error {
	return nil
}
