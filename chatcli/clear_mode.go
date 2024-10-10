package chatcli

import (
	"fmt"

	"github.com/wricardo/code-surgeon/api"
)

func init() {
	RegisterMode("clear", NewClearMode)
}

type ClearMode struct {
	chat *ChatImpl
}

// NewClearMode creates a new instance of the ClearMode, it's executed when the clear mode is activated.
func NewClearMode(chat *ChatImpl) *ClearMode {
	return &ClearMode{chat: chat}
}

// Start is called when the clear mode is activated for interactive mode.
// It clears the chat history and informs the user.
func (m *ClearMode) Start() (*api.Message, *api.Command, error) {
	return m.HandleResponse(&api.Message{})
}

func (m *ClearMode) BestShot(msg *api.Message) (*api.Message, *api.Command, error) {
	return m.HandleResponse(msg)
}

func (m *ClearMode) HandleIntent(msg *api.Message, intent Intent) (*api.Message, *api.Command, error) {
	return m.HandleResponse(msg)
}

func (m *ClearMode) HandleResponse(msg *api.Message) (*api.Message, *api.Command, error) {
	// Clear the chat history
	err := m.chat.ClearHistory()
	if err != nil {
		return &api.Message{Text: fmt.Sprintf("Error clearing chat history: %v", err)}, SILENT_MODE_QUIT_CLEAR, err
	}
	// No additional response handling needed.
	return &api.Message{Text: "history cleared!"}, SILENT_MODE_QUIT_CLEAR, nil
}

func (m *ClearMode) Name() string {
	return "clear"
}

func (m *ClearMode) Stop() error {
	// No cleanup necessary for clear mode.
	return nil
}
